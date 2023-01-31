insert into gaea_collect_config(gmt_create, gmt_modified, tenant, table_name, json, version, collect_range, executor_selector, `type`, deleted) values(now(), now(), 'dev', 'httpcheck_foobar_normal',
'{"executeRule":{"type":"fixedRate","fixedRate":5000},"schema":"http","port":8080,"path":"/path","timeout":3000,"method":"GET","successCodes":[200],"networkMode":"POD"}',
1,
'{"type":"cloudmonitor","cloudmonitor":{"table":"dev_server","condition":[{"app":["cloudmonitor-registry"]}]}}',
'{"sidecar":{},"type":"sidecar"}', 'httpcheck', 0)

