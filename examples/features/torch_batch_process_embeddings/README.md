# Example: use torch_batch_process for embedding generation

## Use case

One common LLM use case is information retrieval. The first step of information retrieval is to generate embedding on
documents and upload to a vector database. 

When a user query come in, the system would first compute embedding of the query, then retrieve documents most relevant
to the query from the vector database. These documents are often used to supplement the prompt provided to the LLM to 
improve the quality of answer.

## How does torch_batch_process API help?

In this example, we use the `torch_batch_process` API to :
1. generate document embeddings across 2 workers,
2. chief worker uploads all the embeddings generated to a local Chroma vector database
    - We upload only via chief worker because Chroma recommends using a single Chroma client at a time 
   ([link](https://docs.trychroma.com/usage-guide) as of 17th July, 2023).

This example can be easily adapted to be used with other vector databases.

## How to run the example

`det e create distributed.yaml .`

## System requirements

This example should be run on an agent machine with at least 2 GPUs. This is because for simplicity, we use the agent's 
file system to store embedding files and to initialize vector database. If the workers are split across machines, this
example would not run properly. 

Alternatively, you can modify the example to use a shared file system directory bind-mounted to the agent machine and 
experiment container. In this case, the number of GPU per machine no longer matters as all agents would have access
to the same directory on the shared file system.
