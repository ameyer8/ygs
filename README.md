# ygs

You Got Served

This tool helps with development by standing up an arbitary directory server and an endpoint to post/retrieve data

    ygs serve

By default this will serve the current directory and subdirecties via HTTP on port 8000

## Dynamic content

POST'ing to the /dyn/$URI endpoint will store arbitry data that can then be retrieved. Once an URI is used, the content can be modified with a PUT operation