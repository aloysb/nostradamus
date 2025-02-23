# Current task goal
Taking into account the CONVENTIONS file, implement the predictions part of the script.
It consistes of receiving an input and generating predictions about upcoming events and their impact on the stock market.

## Task details

CREATE the main.go and main_test.go files.

IMPLEMENT the call to the LLM to generate the predictions. Assume that the api key will be provided via an environment variable. Use open ai o1-mini model. Use a system prompt propice to generating predictions based on an event.

IMPLEMENT the validation step which validate the output of the result:

    - If the output is conform to the output-structure, output it.

    - If the output is not conform, retry up to 10 times

    - If the outpout is still not conform after 10 retry, return an error code and message.

IMPLEMENT all unit tests necessary to thoroughly test the script. Think about all the scenarios possible given the parameters. Ask yourself: "how could this fails?" and generate the test covering the scenario. For example, one parameter is the calling the API: "What if the API is down? Then the script should return an error related to the API down. I will implement a test covering this scenario". Another example would be: "What if the user doesn't provide an input?"

ADD Debugging information. Ensure that all results and steps are logged directly to the std input. Consider that the script will be run in a safe environment and prefer a verbose output, with multiple logs along th way to facilitate debugging.
