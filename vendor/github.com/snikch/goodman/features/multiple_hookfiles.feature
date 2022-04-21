Feature: Multiple hook files with a glob

  Background:
    Given I have "go" command installed
    When I run `go build -o bin/goodman github.com/snikch/goodman/cmd/goodman`
    And I have "dredd" command installed
    And a file named "server.rb" with:
      """
      require 'sinatra'
      get '/message' do
        "Hello World!\n"
      end
      """

    And a file named "apiary.apib" with:
      """
      # My Api
      ## GET /message
      + Request (text)

              This prevents a dredd bug

      + Response 200 (text/html;charset=utf-8)

              Hello World!
      """

  @debug
  Scenario:
    Given a file named "1/hookfile1.go" with:
      """
      package main
      import (
        "fmt"

        "github.com/snikch/goodman/hooks"
        trans "github.com/snikch/goodman/transaction"
      )

      func main() {
          h := hooks.NewHooks()
          server := hooks.NewServer(hooks.NewHooksRunner(h))
          h.Before("/message > GET", func(t *trans.Transaction) {
            fmt.Println("It's me, File1")
          })

        server.Serve()
        defer server.Listener.Close()
      }
      """
    When I run `go build -o 1/hookfile1 github.com/snikch/goodman/tmp/aruba/1`
    And a file named "2/hookfile2.go" with:
      """
      package main
      import (
        "fmt"

        "github.com/snikch/goodman/hooks"
        trans "github.com/snikch/goodman/transaction"
      )

      func main() {
          h := hooks.NewHooks()
          server := hooks.NewServer(hooks.NewHooksRunner(h))
          h.Before("/message > GET", func(t *trans.Transaction) {
            fmt.Println("It's me, File2")
          })

        server.Serve()
        defer server.Listener.Close()
      }
      """
    When I run `go build -o 2/hookfile2 github.com/snikch/goodman/tmp/aruba/2`
    And a file named "hookfile_to_be_globed.go" with:
      """
      package main
      import (
        "fmt"

        "github.com/snikch/goodman/hooks"
        trans "github.com/snikch/goodman/transaction"
      )

      func main() {
          h := hooks.NewHooks()
          server := hooks.NewServer(hooks.NewHooksRunner(h))
          h.Before("/message > GET", func(t *trans.Transaction) {
            fmt.Println("It's me, File3")
          })

        server.Serve()
        defer server.Listener.Close()
      }
      """
    # When I run `go build -o hook_file_to_be_globed github.com/snikch/goodman/tmp/aruba`
    When I run `dredd ./apiary.apib http://localhost:4567 --server "ruby server.rb" --language bin/goodman --hookfiles ./1/hookfile1 --hookfiles ./2/hookfile2`
    Then the exit status should be 0
    And the output should contain:
      """
      It's me, File1
      """
    And the output should contain:
      """
      It's me, File2
      """
    # And the output should contain:
    #   """
    #   It's me, File3
    #   """
