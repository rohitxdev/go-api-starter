# Golang API starter

<img src="./assets/public/images/go-fast.png" alt="Gopher flash" height="128" width="128"/>

## Development

Run the development server:

```bash
./run watch # or ./run docker-watch
```

## Production

Build the project:

```bash
./run build
```

Run the production server:

```bash
./run start
```

## Notes

- The `run` script is used to automate common development/production tasks. Run `./run` to see the available tasks.
- The config/secrets should be stored in a `config.json` file in the root directory. If you want to use a different file, you can specify it using the `CONFIG_FILE` environment variable.
