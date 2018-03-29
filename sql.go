package pushpop

const (
	sqlCreateMessages = `
		CREATE TABLE pushpop_messages (
			id uuid PRIMARY KEY,
			topic text NOT NULL,
			state smallint NOT NULL,
			state_time timestamptz NOT NULL,
			payload text
		)
	`
	sqlIndexMessagesReady = `
		CREATE INDEX pushpop_messages_ready
		ON pushpop_messages (topic, state, state_time)
		WHERE state = 0
	`
	sqlIndexMessagesPending = `
		CREATE INDEX pushpop_messages_pending
		ON pushpop_messages (topic, state, state_time)
		WHERE state = 1
	`

	sqlPushMessage = `
		INSERT INTO pushpop_messages
			(id, topic, state, state_time, payload)
		VALUES
			($1, $2, $3, $4, $5)
	`

	sqlPopMessage = `
		UPDATE pushpop_messages
		SET state = 1, state_time = (now() + interval '5 minutes')
		WHERE id = (
			SELECT id
			FROM pushpop_messages
			WHERE topic = $1
			AND state = 0
			ORDER BY state_time ASC
			FOR UPDATE
			LIMIT 1
		)
		RETURNING id, topic, state, state_time, payload;
	`

	sqlFindMessage = `
		SELECT id, topic, state, state_time, payload
		FROM messages
		WHERE id = $1
	`

	sqlTransitionMessage = `
		UPDATE pushpop_messages
		SET state = $2,
				state_time = $3
		WHERE id = $1
	`
)
