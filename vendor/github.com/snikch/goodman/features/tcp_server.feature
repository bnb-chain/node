Feature: TCP server and messages
  Background:
    When I run `go build -o bin/goodman github.com/snikch/goodman/cmd/goodman`

Scenario: TCP server
  When I run `bin/goodman` interactively
  And I wait for output to contain "Starting"
  Then It should start listening on localhost port "61321"

Scenario: Message exchange for event beforeEach
  Given I run `bin/goodman` interactively
  When I wait for output to contain "Starting"
  And I connect to the server
  And I send a JSON message to the socket:
    """
    {"event": "beforeEach", "uuid": "1234-abcd", "data": {"skip":true}}
    """
  And I send a newline character as a message delimiter to the socket
  Then I should receive same response
  And I should be able to gracefully disconnect

Scenario: Message exchange for event beforeEachValidation
  Given I run `bin/goodman` interactively
  When I wait for output to contain "Starting"
  And I connect to the server
  And I send a JSON message to the socket:
    """
    {"event": "beforeEachValidation", "uuid": "2234-abcd", "data": {"skip":false}}
    """
  And I send a newline character as a message delimiter to the socket
  Then I should receive same response
  And I should be able to gracefully disconnect

Scenario: Message exchange for event afterEach
  Given I run `bin/goodman` interactively
  When I wait for output to contain "Starting"
  And I connect to the server
  And I send a JSON message to the socket:
    """
    {"event": "afterEach", "uuid": "3234-abcd", "data": {"skip":true}}
    """
  And I send a newline character as a message delimiter to the socket
  Then I should receive same response
  And I should be able to gracefully disconnect

Scenario: Message exchange for event beforeAll
  Given I run `bin/goodman` interactively
  When I wait for output to contain "Starting"
  And I connect to the server
  And I send a JSON message to the socket:
    """
    {"event": "beforeAll", "uuid": "4234-abcd", "data": [{"skip":false}]}
    """
  And I send a newline character as a message delimiter to the socket
  Then I should receive same response
  And I should be able to gracefully disconnect

Scenario: Message exchange for event afterAll
  Given I run `bin/goodman` interactively
  When I wait for output to contain "Starting"
  And I connect to the server
  And I send a JSON message to the socket:
    """
    {"event": "afterAll", "uuid": "5234-abcd", "data": [{"skip":true}]}
    """
  And I send a newline character as a message delimiter to the socket
  Then I should receive same response
  And I should be able to gracefully disconnect
