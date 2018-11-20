YELLOW='[93m'
RESET='[0m'

def run(cmdline)
  puts "#{YELLOW}+#{cmdline}#{RESET}"
  system cmdline
end

guard :shell do
  watch /\.go$/ do |m|
    puts "#{Time.now}: #{m[0]}"
    case m[0]
    when /_test\.go$/
      parent = File.dirname m[0]
      sources = Dir["#{parent}/*.go"].reject{|p| p.end_with? '_test.go' }.uniq.join ' '
      # Assume that https://github.com/rhysd/gotest is installed
      run "gotest #{sources}"
      # run "golint #{m[0]}"
    else
      run 'go build ./cmd/trygo'
      # run "golint #{m[0]}"
    end
  end
end
