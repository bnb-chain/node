require 'json'
require 'socket'


Given(/^I have "([^"]*)" command installed$/) do |command|
  is_present = system("which #{ command} > /dev/null 2>&1")
  raise "Command #{command} is not present in the system" if not is_present
end

Given(/^server under test is running$/) do
end

Then(/^It should start listening on localhost port "([^"]*)"$/) do |port|
  @client = TCPSocket.new 'localhost', port
  @client.close
end

Given(/^I connect to the server$/) do
  @client = TCPSocket.new 'localhost', 61321
end

When(/^I send a JSON message to the socket:$/) do |string|
  @data_sent = string
  @client.send @data_sent, 0
end

When(/^I send a newline character as a message delimiter to the socket$/) do
  @client.send "\n", 0
end

Then(/^I should receive same response$/) do
  sleep 1
  data_received = @client.readline
  if JSON.parse(data_received) != JSON.parse(@data_sent)
    @client.close
    raise "Data received:\n#{data_received}\nDoesn't match data sent: #{@data_sent}\n"
  end
end

Then(/^I should be able to gracefully disconnect$/) do
  @client.close
end
