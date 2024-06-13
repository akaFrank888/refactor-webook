-- key取决于具体业务
local key = KEYS[1]
-- 阅读数/点赞数/收藏数
local cntKey = ARGV[1]
-- cntKey对应值的变化量
local delta = tonumber(ARGV[2])

-- 判断redis中该key是否存在
local exist = redis.call("EXISTS", key)
if exist == 1 then
    -- redis的 HINCRBY 命令能够保证，如果 read_cnt 不存在，就先设置为 0，而后自增 1
    redis.call("HINCRBY", key, cntKey, delta)
    return 1
else
    return 0
end