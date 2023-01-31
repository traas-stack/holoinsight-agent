```sql

insert into gaea_collect_config(gmt_create, gmt_modified, tenant, table_name, json, version, collect_range, executor_selector, `type`,
                                deleted)

values (now(), now(),
        'foo',
        'loganalysis_1',
        '{"select":{"values":[{"as":"loganalysis","agg":"loganalysis"}]},"from":{"type":"log","log":{"path":[{"type":"path","pattern":"/home/admin/logs/holoinsight-server/common-default.log"}],"charset":"utf-8","time":{"type":"auto"}}},"where":{},"groupBy":{"logAnalysis":{"patterns":[{"name":"RegistryServiceForAgentImpl","where":{"contains":{"elect":{"type":"line"},"value":"RegistryServiceForAgentImpl"}}},{"name":"DimDataWriteTask","where":{"contains":{"elect":{"type":"line"},"value":"DimDataWriteTask"}}}]}},"window":{"interval":5000},"output":{"type":"gateway","gateway":{"metricName":"loganalysis_1"}}}',
        2,
        '{"type":"cloudmonitor","cloudmonitor":{"table":"foo_server","condition":[{"app":"holoinsight-server"}]}}',
        '{"type":"sidecar","sidecar":{}}', NULL, 0);


update gaea_collect_config
set deleted=1,
    gmt_modified=now()
where table_name = 'loganalysis_1'
  and deleted = 0 limit 1;

```
