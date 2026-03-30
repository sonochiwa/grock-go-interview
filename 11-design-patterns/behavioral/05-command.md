# Command

## В Go

Command инкапсулирует действие как объект. В Go — интерфейс или функция.

```go
type Command interface {
    Execute() error
    Undo() error
}

type AddUserCommand struct {
    repo *UserRepo
    user User
}

func (c *AddUserCommand) Execute() error {
    return c.repo.Add(c.user)
}

func (c *AddUserCommand) Undo() error {
    return c.repo.Delete(c.user.ID)
}

// Command queue
type CommandQueue struct {
    history []Command
}

func (q *CommandQueue) Execute(cmd Command) error {
    if err := cmd.Execute(); err != nil {
        return err
    }
    q.history = append(q.history, cmd)
    return nil
}

func (q *CommandQueue) Undo() error {
    if len(q.history) == 0 {
        return errors.New("nothing to undo")
    }
    last := q.history[len(q.history)-1]
    q.history = q.history[:len(q.history)-1]
    return last.Undo()
}
```
