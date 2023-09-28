package ft

/*
apis to

- receive alert from harness (or internal master calls)
	- validate user has authority over the task: ft.UserOwnsTask
- optional: let users clear alerts they can do this via a new task id.
	- in case something goes wrong and they need to get the same task id scheduled on the same node. can this happen?
- return resolved patterns to watch and actions for harness. it does seem odd to send the action to
harness only to receive it back via alerts :thinking:

*/
