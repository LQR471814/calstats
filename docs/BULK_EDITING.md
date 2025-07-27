# Bulk Editing

Bulk edits are performed with a lua script.

The script should perform mutations on the global variable `events` which is a table of `Event`.

## Event

| Field | Type | Description |
| --- | --- | --- |
| `name` | `string` | Name of the event. |
| `location` | `string` | Location of the event. |
| `description` | `string` | Description of the event. |
| `tags` | `[]string` | Tags on the event. |
| `start` | [DateTime](#DateTime) | Starting time of the event. |
| `end` | [DateTime](#DateTime) | Ending time of the event. |
| `reminder` | [Reminder](#Reminder) | Reminder settings for the event. |

## DateTime

## Reminder

| Field | Type | Description |
| --- | --- | --- |
| `relative` | `number | nil` |

