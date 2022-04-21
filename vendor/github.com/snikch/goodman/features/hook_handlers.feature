Feature: Hook handlers

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

              test this

      + Response 200 (text/html;charset=utf-8)

              Hello World!
      """

  @debug
  Scenario:
    Given a file named "hookfile.go" with:
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
          h.BeforeAll(func(t []*trans.Transaction) {
            fmt.Println("before all hook handled")
          })
          h.BeforeEach(func(t *trans.Transaction) {
            fmt.Println("before each hook handled")
          })
          h.Before("/message > GET", func(t *trans.Transaction) {
            fmt.Println("before hook handled")
          })
          h.BeforeEachValidation(func(t *trans.Transaction) {
            fmt.Println("before each validation hook handled")
          })
          h.BeforeValidation("/message > GET", func(t *trans.Transaction) {
            fmt.Println("before validation hook handled")
          })
          h.After("/message > GET", func(t *trans.Transaction) {
            fmt.Println("after hook handled")
          })
          h.AfterEach(func(t *trans.Transaction) {
            fmt.Println("after each hook handled")
          })
          h.AfterAll(func(t []*trans.Transaction) {
            fmt.Println("after all hook handled")
          })
          server.Serve()
          defer server.Listener.Close()
      }


      """
    When I run `go build -o aruba github.com/snikch/goodman/tmp/aruba`

    When I run `dredd ./apiary.apib http://localhost:4567 --server "ruby server.rb" --language bin/goodman --hookfiles ./aruba --level silly`
    Then the exit status should be 0
    Then the output should contain:
      """
      before hook handled
      """
    And the output should contain:
      """
      before validation hook handled
      """
    And the output should contain:
      """
      after hook handled
      """
    And the output should contain:
      """
      before each hook handled
      """
    And the output should contain:
      """
      before each validation hook handled
      """
    And the output should contain:
      """
      after each hook handled
      """
    And the output should contain:
      """
      before all hook handled
      """
    And the output should contain:
      """
      after all hook handled
      """
