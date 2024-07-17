local zsetName = KEY[1]
local member2incr = ARGV[1]
local newScore = ARGV[2]  -- 指定的分数，如果未指定则默认为1

local exists = redis.call("EXISTS", zsetName)

if exists == 0 then
    redis.call("ZADD", zsetName, newScore, member2incr)
end


-- 获取指定元素的当前分数
local currentScore = redis.call("ZSCORE", zsetName, member2incr)

if currentScore then
    -- 如果元素存在，将分数加1
    local newScore = newScore + 1
    redis.call("ZADD", zsetName, newScore, member2incr)
else
    -- 如果元素不存在，将分数设置为指定的分数
    redis.call("ZADD", zsetName, newScore, member2incr)
end