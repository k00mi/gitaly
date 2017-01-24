def run!(cmd, chdir='.')
  print_cmd = cmd.join(' ')
  puts "-> #{print_cmd}"
  if !system(*cmd, chdir: chdir)
    abort "command failed: #{print_cmd}"
  end
end
