### todo app

This is the simplest example of a graphql server.

to run this server
```bash
go run ./example/todo/server/server.go
```

and open http://localhost:8081 in your browser



# Write your query or mutation here
{
  todos{
    id
    text
    done
  }
  first: todo(id:1){
    id
    text
    done    
  }
  second: todo(id:2){
    id
    text
    done    
  }
  
}