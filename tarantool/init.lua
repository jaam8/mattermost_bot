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
            {name = 'votes',      type = 'string'},
            {name = 'creator_id', type = 'string'},
            {name = 'is_active',  type = 'boolean'},
        }
    })
    polls_space:create_index('primary', {
        if_not_exists = true,
        type = 'tree',
        parts = {1, 'string'}
    })

    local votes_space = box.schema.space.create('votes', {
    if_not_exists = true,
    format = {
        {name = 'poll_id',   type = 'string'},
        {name = 'user_id',   type = 'string'},
        {name = 'choice_id', type = 'string'}
    }})

    votes_space:create_index('primary', {
        if_not_exists = true,
        type = 'hash',
        parts = {'poll_id', 'user_id'}
    })
    votes_space:create_index('user_poll', {
        if_not_exists = true,
        type = 'hash',
        parts = {1, 'string', 2, 'string'}
    })
end)

local log = require('log').new(app_name)
log.info('loaded')
log.info("Tarantool is up and running!")
