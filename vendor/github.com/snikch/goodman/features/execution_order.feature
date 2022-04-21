Feature: Execution order

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

  @announce
  Scenario:
    Given a file named "hookfile.go" with:
    """
    package main
    import (
      "github.com/snikch/goodman/hooks"
      trans "github.com/snikch/goodman/transaction"
    )

    func main() {
        h := hooks.NewHooks()
        server := hooks.NewServer(hooks.NewHooksRunner(h))
        h.BeforeAll(func(t []*trans.Transaction) {
            if t[0].TestOrder == nil {
                t[0].TestOrder = []string{"before all modification"}
                } else {
                    t[0].TestOrder = append(t[0].TestOrder, "before all modification")
                }
        })
        h.BeforeEach(func(t *trans.Transaction) {
            if t.TestOrder == nil {
                t.TestOrder = []string{"before each modification"}
                } else {
                    t.TestOrder = append(t.TestOrder, "before each modification")
                }
        })
        h.Before("/message > GET", func(t *trans.Transaction) {
            if t.TestOrder == nil {
                t.TestOrder = []string{"before modification"}
                } else {
                    t.TestOrder = append(t.TestOrder, "before modification")
                }
        })
        h.BeforeEachValidation(func(t *trans.Transaction) {
            if t.TestOrder == nil {
                t.TestOrder = []string{"before each validation modification"}
                } else {
                    t.TestOrder = append(t.TestOrder, "before each validation modification")
                }
        })
        h.BeforeValidation("/message > GET", func(t *trans.Transaction) {
            if t.TestOrder == nil {
                t.TestOrder = []string{"before validation modification"}
                } else {
                    t.TestOrder = append(t.TestOrder, "before validation modification")
                }
        })
        h.After("/message > GET", func(t *trans.Transaction) {
            if t.TestOrder == nil {
                t.TestOrder = []string{"after modification"}
                } else {
                    t.TestOrder = append(t.TestOrder, "after modification")
                }
        })
        h.AfterEach(func(t *trans.Transaction) {
            if t.TestOrder == nil {
                t.TestOrder = []string{"after each modification"}
                } else {
                    t.TestOrder = append(t.TestOrder, "after each modification")
                }
        })
        h.AfterAll(func(t []*trans.Transaction) {
            if t[0].TestOrder == nil {
                t[0].TestOrder = []string{"after all modification"}
                } else {
                    t[0].TestOrder = append(t[0].TestOrder, "after all modification")
                }
        })

        server.Serve()
        defer server.Listener.Close()
    }
    """
    When I run `go build -o aruba github.com/snikch/goodman/tmp/aruba`
    And I set the environment variables to:
      | variable                       | value      |
      | TEST_DREDD_HOOKS_HANDLER_ORDER | true       |

    When I run `dredd ./apiary.apib http://localhost:4567 --server "ruby server.rb" --language bin/goodman --hookfiles ./aruba`
    Then the exit status should be 0
    Then the output should contain:
      """
      0 before all modification
      1 before each modification
      2 before modification
      3 before each validation modification
      4 before validation modification
      5 after modification
      6 after each modification
      7 after all modification
      """
