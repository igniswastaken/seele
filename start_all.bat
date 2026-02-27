@echo off
echo Starting Seele Cluster...

start "Shard 1" cmd /k "go run . -port 8081 -dir ./shards/shard1"
timeout /t 2 >nul

start "Shard 2" cmd /k "go run . -port 8082 -dir ./shards/shard2 -join 127.0.0.1:9081"
timeout /t 2 >nul

start "Shard 3" cmd /k "go run . -port 8083 -dir ./shards/shard3 -join 127.0.0.1:9081"
timeout /t 2 >nul

start "Proxy" cmd /k "go run . -proxy -port 8080 -join 127.0.0.1:9081"

echo Cluster started successfully!
echo Proxy is listening on http://localhost:8080
echo.
pause