problem: fixed nicks probably should not show up in leaderboard entries
but, they should still exist in file layer, because the file records unprocessed information

solution: a config file that merges fixed nicks to user snowflakes

a script, in cmd/<something>, that does:
inits the nick cache, reads the storage file, records all existing fixed nicks
- (if config file does not exist) outputs user id to nick mappings, and fixed nicks that aren't mapped
- (if config file does exist) verifies the config file by making sure all fixed nicks are mapped

then, support for the config file itself, in the query layer. 
