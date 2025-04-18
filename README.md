# **Advanced Subdomain Enumeration Tool: Complete Setup Guide**

## System Requirements:

- Go (Golang) 1.21+



## Required Go Libraries

- git clone github.com/fatih/color
- golang.org/x/sync/errgroup

## 1.Installing Go

1.Install Go using macOS





``` # Using Homebrew
brew install go


go version
```

2. Install Go using Windows
``` 

Download Go installer from official website
Run installer
Set system PATH environment variable
Open PowerShell/CMD and verify:
# Verify installation
Go Version

```

3. Install Go Using Linux (Ubuntu/Debian)

```

wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version

```

*OR*
```
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.zshrc
source ~/.zshrc
go version

```


## 2.Clone the Repository

```

go install -v github.com/ZeeeboX5/UFOScanner@latest
```



## 3.Configuration of Proxy


In the main.go file, you can configure proxies if needed:

```
proxies := []ProxyConfig{
    {
        URL:      "http://proxy1.example.com:8080",
        Protocol: "http",
        Auth: &ProxyAuth{
            Username: "user",
            Password: "pass",
        },
    },
    // Add more proxies as needed
}
```


# Running the Tool












