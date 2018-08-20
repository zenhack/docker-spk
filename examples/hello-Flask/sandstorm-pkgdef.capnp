@0xbd8cfd59c13fc42f;

using Spk = import "/sandstorm/package.capnp";

const pkgdef :Spk.PackageDefinition = (
  # Nothing here is especially important to this example; see the comments in
  # the sandstorm-pkgdef.capnp output by `docker-spk init` for details on what
  # everything does.
  id = "8anwd8gxasmhav7uu869eamag7rqhxczd86a6fcz5nktuk02mkth",

  manifest = (
    appTitle = (defaultText = "Hello World App With Flask and 'docker-spk'"),
    appVersion = 0,
    appMarketingVersion = (defaultText = "0.0.0"),
    actions = [
      ( nounPhrase = (defaultText = "instance"),
        command = .myCommand
      )
    ],
    continueCommand = .myCommand,
    metadata = (
      website = "http://example.com",
      codeUrl = "http://example.com",
      license = (openSource = apache2),
      author = (
        contactEmail = "youremail@example.com",
      ),
      shortDescription = (defaultText = "one-to-three words"),
    ),
  ),
);

const myCommand :Spk.Manifest.Command = (
  # Command to start the server. We point sandstorm-http-bridge at gunicorn,
  # and point gunicorn at our Flask app.
  argv = [
    "/sandstorm-http-bridge", "8000", "--", "/app/.venv/bin/gunicorn", "hello_flask:app"
  ],
  environ = [
    # Note that this defines the *entire* environment seen by your app.
    (key = "PATH", value = "/usr/local/bin:/usr/bin:/bin"),

    (key = "HOME", value = "/var")
    # Work around a bug in python; see
    # https://docs.sandstorm.io/en/latest/developing/raw-python/
  ]
);
