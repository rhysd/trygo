YELLOW='[93m'
RESET='[0m'

ignore %r!^out/!, %r!^testdata/gen/.*/HAVE!, %r!^tmp/!

def run(cmdline)
  puts "#{YELLOW}+#{cmdline}#{RESET}"
  system cmdline
end

def run_tests(file, flags='')
  parent = File.dirname file
  sources = Dir["#{parent}/*.go"].reject{|p| p.end_with? '_test.go' }.uniq.join ' '
  sources = "common_test.go #{sources}" if file != 'common_test.go'
  # Assume that https://github.com/rhysd/gotest is installed
  run "gotest #{flags} #{file} #{sources}"
end

guard :shell do
  watch /\.go$/ do |m|
    puts "#{Time.now}: #{m[0]}"
    case m[0]
    when /_test\.go$/
      run_tests m[0]
    when %r{^testdata/gen/}
      run_tests 'generate_test.go'
    when %r{^testdata/trans/ok/([^/]+)/}
      run_tests 'translate_test.go', "-run TestTranslationOK/#{$1}"
    when %r{^testdata/trans/error/([^/]+)/}
      run_tests 'translate_test.go', "-run TestTranslationError/#{$1}"
    when %r{^[^/]+\.go$}
      run 'go build ./cmd/trygo'
      run "golint #{m[0]}"
      run 'go vet'
    end
  end

  watch /\.txt$/ do |m|
    puts "#{Time.now}: #{m[0]}"
    case m[0]
    when %r{^testdata/trans/error/([^/]+)/message\.txt$}
      run_tests 'translate_test.go', "-run TestTranslationError/#{$1}"
    end
  end
end
