# Re-organize the user guide

Re-organise doc/content in JTBD style, with examples, keeping all the essential content intact.
Keep a strong reference section, link to it to avoid over-complicating examples.
The non-reference section should allow readers to get started quickly and easily and understand the possibilities, but we can't provide examples for every possibility. Links to the reference section allow the user to explore what can be done beyond the examples.

The guide should include
- getting started: installation and basic use concise but clear for various scenarios:
  - installing as a cluster service via COO (main scenario)
  - using the command from outside the cluster (development & experiments)
  - running as a service outside the cluster (development)
- using the korrel8rcli client -  link to reference docs on korrel8rcli site, don't copy them
- introduction/concepts
  - short but necessary, korrel8r concepts require some explanation.
- Openshift troubleshooting panel
  - brief intro with diagram, pointer to official docs
  - explain this is one important application of korrel8r but others are possible
- AI agents
- Writing rules (new section)
- Other uses (new section, explore this idea)
  - talk about other uses for korrel8r

References to consult:
- this directory - you can check the code for correctness of doc.
- ../korrel8rcli/doc - client documentation
- lab/mcp/ - AI setup and use
- ../demos/agent-console/ - AI demo
- ../troubleshooting-panel-console-plugin_user-guide/doc - Openshift console

Feel free to propose alternate structures, I'm exploring better ways to present the material
