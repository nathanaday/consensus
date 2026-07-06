# Phase 2

We now have some basic foundation for loading time series data from csv files, and a few extra tools for the user to get oriented with their time series data. Once more, before we dive into the analysis, we should flesh out this idea of "data hierarchy" as data is copied and modified.

When the user performs a statistical operation on a dataset, a copy of that dataset is created. Data is never modified in place at this stage. Over time, you can think of all datasets as a graph, with each dataset being a node, and an edge created whenever something is spawned and mutated from that dataset. We need to maintain this graph and gain the ability to traverse it.

I suggest we implement a basic approach where we keep a list of root nodes. These are just new datasets loaded from csv. Anytime a dataset is loaded, it is added to the root node list. 

Then, when a dataset is copied and modified, the root node needs to create a pointer to the copied object. It is possible to copy many datasets from each root node, so a list of pointers is more appropriate.

To store this in memory, we can save a simple adjaceny list, either JSON based or just text based. When loaded into memory, it should be the object + pointer model paradigm. 

When the objects are loaded, it is now possible to gain valuable insights like: 
- what dataset was this layer copied from?
- show me all datasets that are created from this dataset

The app also needs instructions to also make a copy and update the node graph when making changes to the data. We should design a dedicated utility layer for managing this copy with a super easy SDK or API so the actual MCP tool implementation does not need to concern itself with the inner workings of this node graph.

It should just be able to use a syntax loosely based on this pseudocode:

```
child = dataset.copy()
parent = child.get_parent()
grandparent = parent.parent()
new_parent = grandparent.copy()

children_list = parent.get_children()

info = child.get_info() // returns the dataset name, size, units, filepath, etc...

child_data = child.load_data() // loads the timeseries data for analysis work
```

We would also like to output the full DAG in a visual format. I suggest mermaid.js



## Example Queries


> what dataset was <name> copied from?

> show me all datasets that were created from the <name> dataset

> create a new copy of the <name> dataset

> output the full datagraph of all datasets we have

> output the mermaid.js showing all datasets we have









