## Runtime Detection Algorithm

This section describes how ad-runtime-utils chooses the correct installation path for a given runtime (Java, Python, etc.), both when a service is specified and in the default (no-service) case.

---

### 1. Service-Specific Flow (`--service <NAME>`)

1. **Service override_path**
    - If `services.<NAME>.runtimes.<RT>.override_path` is set and `<path>/bin/<exe>` exists → **return** that path.

2. **Service env_var**
    - If `services.<NAME>.runtimes.<RT>.env_var` is set, and the named environment variable is non-empty and points at a valid `<path>/bin/<exe>` → **return** its value.

3. **Service version**
    - Read `services.<NAME>.runtimes.<RT>.version`.
    - If missing or empty → **error**:
      ```
      version not specified for service '<NAME>' runtime '<RT>'
      ```

4. **Per-version Autodetect** (`autodetect.runtimes.<RT>.<version>`)
    1. If `override_path` is set and valid → **return** it.
    2. If `env_var` is set and points at a valid path → **return** it.
    3. Otherwise, for each glob in `paths` (in order):
        - `candidates := Glob(pattern + "*")`
        - sort in reverse lexical order
        - first candidate with `<cand>/bin/<exe>` → **return** that path.

5. **Fallback to Default-Flow**
    - If none of the above steps succeed, repeat the **Default-Flow** (see below), but format output as:
      ```
      <NAME>: /path/from/default-flow
      ```

6. **Error**
    - If still nothing is found →
      ```
      no <RT> environment found for service '<NAME>' (version '<version>')
      ```

---

### 2. Default Flow (no `--service`)

1. **Default override_path**
    - If `default.runtimes.<RT>.override_path` is set and valid → **return** it.

2. **Default env_var**
    - If `default.runtimes.<RT>.env_var` is set and points at a valid path → **return** it.

3. **Default version**
    - Read `default.runtimes.<RT>.version`.
    - If missing or empty → **error**:
      ```
      default version not specified for runtime '<RT>'
      ```

4. **Per-version Autodetect** (`autodetect.runtimes.<RT>.<version>`)
    - Same 4.1–4.3 as in the Service-Specific Flow.

5. **Error**  

### 3. Listing All Detected Runtimes (--list / -l)

When the --list (or -l) flag is provided:

It outputs all detected runtimes for:

- The default configuration
- Each defined service

```
Default runtimes:
  <runtime>: <path or "error: <message>">

Service <service-name>:
  <runtime>: <path or "error: <message>">
  …
```