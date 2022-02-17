# Cupid

## 简介

+ 借爱神丘比特（`Cupid`）之名，愿天下相爱之人终成眷属，都能抢得`HPV-9`疫苗。

## 指南

### 抓包

### 嗅探

+ 从秒苗（约苗的微信小程序）上获取疫苗的秒杀信息。

```bash
# 自动执行
make run-sniff

# 手动执行
go run main.go sniff -c configs/configs.yaml
```

![秒杀信息](images/seckill.png)

+ 启动约苗小助手，订购疫苗：

```bash
```

+ 后台执行任务：

```bash
./cupid sniff -c configs.yaml
nohup ./cupid seckill -c configs.yaml &
```

+ 约苗的`Token`过期时间为`1`小时。
+ 知苗易约的`Cookie`过期时间为`2`小时。

***
