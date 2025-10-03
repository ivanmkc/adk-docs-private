# Installing ADK

=== "Python"

    ## Create & activate virtual environment

    We recommend creating a virtual Python environment using
    [venv](https://docs.python.org/3/library/venv.html):

    ```shell
    python -m venv .venv
    ```

    Now, you can activate the virtual environment using the appropriate command for
    your operating system and environment:

    ```
    # Mac / Linux
    source .venv/bin/activate

    # Windows CMD:
    .venv\Scripts\activate.bat

    # Windows PowerShell:
    .venv\Scripts\Activate.ps1
    ```

    ### Install ADK

    ```bash
    pip install google-adk
    ```

    (Optional) Verify your installation:

    ```bash
    pip show google-adk
    ```

=== "Java"

    You can either use maven or gradle to add the `google-adk` and `google-adk-dev` package.

    `google-adk` is the core Java ADK library. Java ADK also comes with a pluggable example SpringBoot server to run your agents seamlessly. This optional
    package is present as part of `google-adk-dev`.

    If you are using maven, add the following to your `pom.xml`:

    ```xml title="pom.xml"
    <dependencies>
      <!-- The ADK Core dependency -->
      <dependency>
        <groupId>com.google.adk</groupId>
        <artifactId>google-adk</artifactId>
        <version>0.2.0</version>
      </dependency>

      <!-- The ADK Dev Web UI to debug your agent (Optional) -->
      <dependency>
        <groupId>com.google.adk</groupId>
        <artifactId>google-adk-dev</artifactId>
        <version>0.2.0</version>
      </dependency>
    </dependencies>

    <build>
      <plugins>
        <plugin>
          <groupId>org.apache.maven.plugins</groupId>
          <artifactId>maven-compiler-plugin</artifactId>
          <version>3.14.0</version>
          <configuration>
            <compilerArgs>
              <arg>-parameters</arg>
            </compilerArgs>
          </configuration>
        </plugin>
      </plugins>
    </build>
    ```

    Here's a [complete pom.xml](https://github.com/google/adk-docs/tree/main/examples/java/cloud-run/pom.xml) file for reference.

    If you are using gradle, add the dependency to your build.gradle:

    ```title="build.gradle"
    dependencies {
        implementation 'com.google.adk:google-adk:0.2.0'
        implementation 'com.google.adk:google-adk-dev:0.2.0'
    }
    ```

    You should also configure Gradle to pass `-parameters` to `javac`. (Alternatively, use `@Schema(name = "...")`).

=== "Go"

    ## Create a new Go module

    If you are starting a new project, you can create a new Go module:

    ```shell
    go mod init example.com/my-agent
    ```

    ## Install ADK

    To add the ADK to your project, run the following command:

    ```shell
    go get google.golang.org/adk
    ```

    This will add the ADK as a dependency to your `go.mod` file.

    (Optional) Verify your installation by checking your `go.mod` file for the `github.com/google/adk-go` entry.


## Next steps

* Try creating your first agent with the [**Quickstart**](quickstart.md)
