insert into gaea_collect_config(gmt_create, gmt_modified, tenant, table_name, json, version, collect_range, executor_selector, `type`, deleted) values(now(), now(), 'dev', 'dialcheck_foobar_normal',
'{"executeRule":{"type":"fixedRate","fixedRate":5000},"network":"tcp","port":8080,"networkMode":"POD"}',
1,
'{"type":"cloudmonitor","cloudmonitor":{"table":"dev_server","condition":[{"app":["registry"]}]}}',
'{"sidecar":{},"type":"sidecar"}', 'dialcheck', 0)

