package collecttask

import (
	"database/sql"
	"github.com/TRaaSStack/holoinsight-agent/pkg/server/registry/pb"
	"github.com/golang/protobuf/proto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type (
	// 为了防止与 CollectTask 耦合, 我们单独创建一个DO对象用于存储层
	// Collect Task Data Object
	//CollectTaskDO struct {
	//}

	// 存储要考虑的点:
	// 1. 方便扩展, 从这点考虑显然sqlite是最保险的
	// 2. 方便排查问题,sqlite可以直接将db文件下载下来然后打开, 其他kv数据库需要写程序去解析文件才能读取
	// 3. 不会浪费太多磁盘空间, 考虑到我们这是云上的case, 每个agent收到的任务是极少的, 其实不用太担心
	// 3.1 但是如果是站在主站agent去考虑这个事情的话, 那么需要考虑如下:
	// 3.1.1 一个采集任务=采集配置+采集目标, 不同采集任务的相同采集配置和相同采集目标在磁盘里有几个副本? 显然现有的agent/vessel都是多个副本
	// 3.1.2 主站某台vessel有将近6万个采集任务, 它的boltdb数据库文件大小达到400MB
	// 3.1.3 如果进行合理的合并, 保证采集配置/目标相同的只有一个副本其实不会这么大
	// 3.1.4 但进行合并的话会带来实现上的麻烦, 理论上肯定是可以实现的, 但编码起来会比较麻烦, 容易出错

	// 存储方案1: 使用sqlite存储, 每个 CollectTaskDO 对应一条记录, columns 大概有 bucket,task_key,config_key,target_key,data(是个二进制)
	// 存储方案2: 使用boltdb存储, 每个buckets自己一个bucket(boltdb的概念), 以 task_key 作为 boltdb 的 key, 整个对象序列化后作为 bolt db 的 value; 但bolt年代比较久远了,不太活泼, 起码得用bbolt代替
	// 存储方案3: 使用一般的持久化kv存储, 比如 badgerdb, 但它没有boltdb的bucket的概念(类似namespace), 因此所有buckets的数据是存在一起的
	BucketDO struct {
		ID          int64 `gorm:"primarykey"`
		GmtCreate   time.Time
		GmtModified time.Time
		Name        string `gorm:"unique;"`
		State       string `gorm:"not null;"`
	}

	CollectTaskDO struct {
		ID           int64 `gorm:"primarykey"`
		GmtCreate    time.Time
		GmtModified  time.Time
		Bucket       string `gorm:"index;"`
		Key          string `gorm:"unique;"`
		Version      string `gorm:""`
		CollectBytes []byte `gorm:""`
		TargetBytes  []byte `gorm:""`
	}

	// Collect Task Storage
	Storage struct {
		db *gorm.DB
	}

	// 复用复用配置
	resourceReuse struct {
		configs map[string]*CollectConfig
		targets map[string]*CollectTarget
	}
)

func NewStorage(path string) (*Storage, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&BucketDO{}, &CollectTaskDO{})
	if err != nil {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		return nil, err
	}
	return &Storage{
		db: db,
	}, nil
}

func (s *Storage) GetAll() (map[string]*BucketInfo, error) {
	var bucketDos []*BucketDO
	var taskDos []*CollectTaskDO

	err1 := s.db.Transaction(func(tx *gorm.DB) error {

		if result := tx.Model(&BucketDO{}).Find(&bucketDos); result.Error != nil {
			return result.Error
		}

		if result := tx.Model(&CollectTaskDO{}).Find(&taskDos); result.Error != nil {
			return result.Error
		}

		return nil

	}, &sql.TxOptions{ReadOnly: true})
	if err1 != nil {
		return nil, err1
	}

	buckets := make(map[string]*BucketInfo, len(bucketDos))
	for _, bucketDo := range bucketDos {
		buckets[bucketDo.Name] = &BucketInfo{
			key:   bucketDo.Name,
			state: bucketDo.State,
			tasks: make(map[string]*CollectTask),
		}
	}

	reuse := newResourceReuse()
	for _, taskDo := range taskDos {
		bi := buckets[taskDo.Bucket]
		if bi == nil {
			continue
		}

		pbConfig := &pb.CollectConfig{}
		proto.Unmarshal(taskDo.CollectBytes, pbConfig)

		pbTarget := &pb.CollectTarget{}
		proto.Unmarshal(taskDo.TargetBytes, pbTarget)

		bi.tasks[taskDo.Key] = &CollectTask{
			Key:     taskDo.Key,
			Version: taskDo.Version,
			Config:  reuse.reuseConfig(pbConfig),
			Target:  reuse.reuseTarget(pbTarget),
		}
	}

	return buckets, nil
}

func (s *Storage) Remove(bucket string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if r := tx.Delete(&BucketDO{}, &BucketDO{Name: bucket}); r.Error != nil {
			return r.Error
		}
		if r := tx.Delete(&CollectTaskDO{}, &CollectTaskDO{Bucket: bucket}); r.Error != nil {
			return r.Error
		}
		return nil
	})
}

func (s *Storage) ApplyDelta() error {
	return nil
}

func (s *Storage) SetBucketState(bucket string, state string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {

		now := time.Now()

		bucketDo := BucketDO{
			GmtCreate:   now,
			GmtModified: now,
			Name:        bucket,
			State:       state,
		}

		// 存在就更新
		tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"state"}),
		}).Create(&bucketDo)

		return nil
	})
}

func (s *Storage) Set(bucket *BucketInfo) error {
	return s.db.Transaction(func(tx *gorm.DB) error {

		now := time.Now()

		bucketDo := BucketDO{
			GmtCreate:   now,
			GmtModified: now,
			Name:        bucket.key,
			State:       bucket.state,
		}

		// 存在就更新
		// insert into ... on duplicated update ...
		err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{"state", "gmt_modified"}),
		}).Create(&bucketDo).Error
		if err != nil {
			return err
		}

		var taskDos []*CollectTaskDO
		// select * from buckets where bucket = ?
		err = tx.Where(&CollectTaskDO{Bucket: bucket.key}).Find(&taskDos).Error
		if err != nil {
			return err
		}

		var deleteDos []*CollectTaskDO
		for _, taskDo := range taskDos {
			if _, ok := bucket.tasks[taskDo.Key]; !ok {
				deleteDos = append(deleteDos, taskDo)
			}
		}

		if len(deleteDos) > 0 {
			// delete from buckets where id in ...
			err = tx.Delete(deleteDos).Error
			if err != nil {
				return err
			}
		}

		// 这些需要set
		var setDos []*CollectTaskDO
		for _, task := range bucket.tasks {
			taskDo := CollectTaskDO{
				GmtCreate:    now,
				GmtModified:  now,
				Bucket:       bucket.key,
				Key:          task.Key,
				Version:      task.Version,
				CollectBytes: toPbCollectConfigBytes(task.Config),
				TargetBytes:  toPbCollectTargetBytes(task.Target),
			}
			setDos = append(setDos, &taskDo)
		}

		if len(setDos) > 0 {
			// insert into ... on duplicated update ...
			err = tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "key"}},
				DoUpdates: clause.AssignmentColumns([]string{"version", "collect_bytes", "target_bytes", "gmt_modified"}),
			}).Create(&setDos).Error
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Storage) Close() {
	sqlDB, _ := s.db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}
}

func newResourceReuse() *resourceReuse {
	return &resourceReuse{
		configs: make(map[string]*CollectConfig),
		targets: make(map[string]*CollectTarget),
	}
}

func (r *resourceReuse) reuseConfig(pbConfig *pb.CollectConfig) *CollectConfig {
	c, ok := r.configs[pbConfig.Key]
	if ok {
		return c
	}
	c = toCollectConfig(pbConfig)
	r.configs[c.Key] = c
	return c
}

func (r *resourceReuse) reuseTarget(pbTarget *pb.CollectTarget) *CollectTarget {
	t, ok := r.targets[pbTarget.Key]
	if ok {
		return t
	}
	t = toCollectTarget(pbTarget)
	r.targets[t.Key] = t
	return t
}
