# nvidia-smi commands

list all GPUs
nvidia-smi -L

query GPU,memory utilization
nvidia-smi --query-gpu=index,utilization.gpu,utilization.memory --format=csv

> nvidia-smi --help-query-gpu

query memory usage
nvidia-smi --query-gpu=index,memory.used,memory.free,memory.total --format=csv

query version and driver
nvidia-smi --query-gpu=index,name,vbios_version,driver_version --format=csv

query temperature,power,clocks
nvidia-smi --query-gpu=index,power.draw,temperature.gpu,clocks.current.sm,clocks.current.memory,fan.speed --format=csv

query GPU used memory of pids
nvidia-smi --query-compute-apps=gpu_uuid,pid,used_memory --format=csv
