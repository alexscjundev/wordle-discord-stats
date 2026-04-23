
## not finishing

starting and not finishing is a score of 7

## scoring features

all of these should map from player to value
and support query of either top k, or bottom k, or one player's value

### total elo

long term elo
treats single day as several 1v1 games
adjust elo, and move to next day

example:

alex: 3
bob: 4
charles: 5
alex v bob: alex W
bob v charles: bob W
alex v charles: alex W

sample starting value: 1500
sample k: 32

### min 20 simple

just take the average score of each player, over all time
then discard the players that have played under 20 games

### sliding week score

over the most recent 7 days
take the average score of each player

## misc features

### current streaks

count current streak of player
used to display in daemon report header

### all time streak

count best streak of player, and when
used as fun fact in daemon report header

### scores <= x

count number of times player got <= x
used as fun fact in daemon report header

## thoughts on implementation

## load
load function will be used to query info, and should sort by a) wordle day then b) player name to guarantee ordering

### decompose it properly, focus on good interfaces

new query type, bottom k. top k only used for total elo

new query param, query type. one per each query above

query will not fix value like "7" for sliding score or "1500" for total elo, rather allow it as provided input to the query, but fix from what we call from daemon 

every query uses "per player view" except total elo, uses "per day view"
so decompose into two helpers that call load()

perPlayerResults, map[string][]wordleResult
perDayResults, map[int][]wordleResult

and call them respectively

### fun facts

only one fun fact per report
50/50 between all time streak or scores <= x
if scores <= x, random number between 2 and 4, then random person from the results
