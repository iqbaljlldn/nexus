-- rate_limit.lua — Sliding window rate limiter via Redis Sorted Set
-- Executed atomically via EVAL/EVALSHA to prevent race conditions
-- between ZCARD check and ZADD increment.
--
-- KEYS[1] = rate limit key (e.g. "ratelimit:login:user@example.com")
-- ARGV[1] = current timestamp in milliseconds
-- ARGV[2] = window size in milliseconds
-- ARGV[3] = maximum allowed requests within the window
--
-- Returns: 1 if allowed, 0 if rate limit exceeded

local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])

redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
local count = redis.call('ZCARD', key)
if count >= limit then
    return 0
end
redis.call('ZADD', key, now, now .. '-' .. math.random())
redis.call('EXPIRE', key, math.ceil(window / 1000))
return 1
