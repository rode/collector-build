# collector-build

A generic build collector for Rode

Build occurrence field assumptions:
- note name is static, not tied to an actual note (`projects/rode/notes/build_collector`)
- omit provenance bytes -- we aren't signing anything at the moment
- build provenance id is random UUID, not tied to CI job that built it
- emtpy commands -- would need greater integration with CI system
- omit startTime/endTime -- would need CI-specific info  
- omitted creator and logs URI -- specific to CI system
- almost all of the sourceProvenance fields, except for `context.git` 
- omit triggerId, buildOptions, builderVersion
