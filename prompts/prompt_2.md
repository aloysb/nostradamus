# Current task goal
Taking into account the CONVENTIONS file, implement the predictions part of the script.
It consistes of receiving an input and generating predictions about upcoming events and their impact on the stock market.

## Task details
UPDATE the payload to the API call to use the field "messages" over "prompt". The filed message should be formatted as: 
"""
messages: [{role:"user", content: "{...}" (content of the prompt here (finalPrompt)}]
