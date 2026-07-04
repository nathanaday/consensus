# Phase 1

Currently the only tool that has been created is the CSV ingest tool. Before we dive into the analysis side, let's make sure the MCP tools have a good foundation for working with the existing data sets. This includes all the tools we would need to interact with the data store and get oriented.


## Example Queries

These queries should work. During brainstorm, we will design the tool set to accomodate these example prompts.

### Ask about existing data

The tool should be able to fetch all datasets and their metadata. Metadata includes the dataset name, information about the data type, size, and time range.

> What datasets have we loaded?

> What is the time range of the <name> dataset?

> How many values are stored in <name> dataset?

> What are the units of measurement in the <name> dataset?

### Ask about the application settings and capabilities

Should be able to fetch the current data store location and a list of files inside this location. The tool should also be able to access a static set of capabilities and features. These can be updated as the tool progreses, but should remain accurate and realistic.

> Where is my data stored?

> In what file format is my data stored

> What file formats does this app support


