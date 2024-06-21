
.PHONY: run bundle

run:
	source ./set_env.sh && go run .

bundle:
	$(MAKE) -C bundle -f Makefile clean
	$(MAKE) -f bundle/Makefile