
.PHONY: run bundle

run:
	source ./set_env.sh && go run .

commit:
	source ./set_env.sh && git cz

bundle:
	$(MAKE) -C bundle -f Makefile clean
	$(MAKE) -f bundle/Makefile