# brutessh
Use Golang developed one tiny SSH bruter

### Compile
```
go get -u github.com/huy4ng/brutessh
go build brute.go
```

### Useage
```
  -P string
    	File with passwords
  -T int
    	Amount of worker working concurrently (default 5)
  -U string
    	File with usernames
  -ip string
    	IP of the SSH server
  -port string
    	Port of the SSH server (default "22")
  -timeout int
    	Timeout per connection in seconds (default 5)
  -u string
    	SSH user to bruteforce
```
 ### For Example
 ```
 ./brute -T 10 -P password.txt -ip 127.0.0.1 -u root
 ./brute -T 10 -P password.txt -ip 127.0.0.1 -U username.txt
 ```

### TODO

- Support ip list brute
- Support one password brute