from setuptools import setup

setup(name='hello-flask-docker-spk',
      version='0.1',
      description='Hello world sandstorm app using flask and docker-spk',
      py_modules=[
          'hello_flask',
      ],
      install_requires=[
          'Flask>=1.0,<2.0',
      ])
