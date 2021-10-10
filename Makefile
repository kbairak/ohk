clean:
	rm -rf build dist


build: clean
	python -m build


publish: build
	python -m twine upload dist/*
