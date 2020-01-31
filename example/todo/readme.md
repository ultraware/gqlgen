## todo app

This is the simplest example of a graphql server.

to run this server
```bash
go run ./example/todo/server/server.go
```

and open http://localhost:8081 in your browser



## example query
```
{
  todos{
    id
    text
    done
    sub {
      text
    }
  }
  first: todo(id:1){
    id
    text
    done    
    sub {
      text
    }
  }
  second: todo(id:2){
    id
    text
    done    
    sub {
      text
    }
  }
}
```