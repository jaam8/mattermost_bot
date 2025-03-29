local app_name = "mattermost_bot"

box.cfg{
    listen = os.getenv("TARANTOOL_PORT")
}

box.schema.user.create(os.getenv("TARANTOOL_USER"), {password = os.getenv("TARANTOOL_PASSWORD"), if_not_exists=true})
box.schema.user.grant(os.getenv("TARANTOOL_USER"), 'super', nil, nil, {if_not_exists=true})

box.once('bootstrap_schema', function()
    local polls_space = box.schema.space.create('polls', {
        if_not_exists = true,
        format = {
            {name = 'id',         type = 'string'},
            {name = 'question',   type = 'string'},
            {name = 'options',    type = 'array'},
            {name = 'votes',      type = 'map'},
            {name = 'creator_id', type = 'string'},
            {name = 'active',     type = 'boolean'},
        }
    })
    polls_space:create_index('primary', {
        if_not_exists = true,
        type = 'hash',
        parts = {1, 'string'}
    })

    local votes_space = box.schema.space.create('votes', {
    if_not_exists = true,
    format = {
        {name = 'poll_id', type = 'string'},
        {name = 'user_id', type = 'string'},
        {name = 'choice', type = 'integer'}
    }})

end)

local log = require('log').new(app_name)
log.info('loaded')
print("Tarantool is up and running!")
local result = box.space.polls:select({}, {limit = 10})
print(result)
