# RefactAI

Agente de refactorización de código escrito en **Go**, basado en el paradigma **CodeAct (code-as-action)**: en lugar de responder con texto o llamadas a funciones JSON, el LLM genera un **programa Go ejecutable** que se compila y corre para modificar el proyecto.

RefactAI **no es un agente generalista**. Está especializado en detectar problemas concretos de calidad de código (funciones demasiado largas, funciones con demasiados parámetros, etc.) mediante un analizador estático, y usa esos hallazgos como guía principal para que el LLM proponga y ejecute cambios mínimos y explicables — no para rediseñar el proyecto libremente.

---

## Tabla de contenidos

- [¿Qué hace?](#qué-hace)
- [Code as Action](#code-as-action)
- [Arquitectura y flujo](#arquitectura-y-flujo)
- [Aislamiento y seguridad](#aislamiento-y-seguridad)
- [Requisitos](#requisitos)
- [Instalación](#instalación)
- [Configuración](#configuración)
- [Uso](#uso)
- [Ejemplo](#ejemplo)
- [Tests](#tests)
- [Estado del proyecto](#estado-del-proyecto)
- [Qué falta / próximos pasos](#qué-falta--próximos-pasos)
- [Decisiones de diseño](#decisiones-de-diseño)

---

## ¿Qué hace?

Dado un proyecto Go y una tarea de refactorización en lenguaje natural, RefactAI:

1. Analiza el código con un **Analyzer** basado en `go/ast` que detecta problemas estructurales (funciones muy largas, funciones con demasiados parámetros).
2. Usa esos *findings* junto con la tarea del usuario para que un LLM (Gemini) genere un **plan de implementación**.
3. Genera un **programa Go ejecutable** (una "Action") que aplica ese plan sobre una copia aislada del proyecto (el *workspace*).
4. Compila y ejecuta esa Action, valida el resultado (`go test ./...` si aplica) y, si algo falla, retroalimenta el error al LLM para que corrija (hasta 3 intentos).
5. Compara el proyecto original contra el workspace modificado y muestra un **diff**.
6. Solo si el usuario **aprueba explícitamente**, los cambios se aplican sobre el proyecto real. Si se rechazan, el workspace temporal se descarta y el proyecto original queda intacto.

---

## Code as Action

El LLM no devuelve un diff ni una lista de ediciones: devuelve un archivo `action.go` completo, con `package main` y `func main()`, que se compila junto a un segundo archivo fijo (`refactai_tools.go`, no generado por el LLM) que expone primitivas seguras y deterministas:

```go
func replaceExact(path, oldStr, newStr string) error
func replaceFunction(path, functionName, newSource string) error
func createFile(path, content string) error
func deleteFile(path string) error
```

- `replaceFunction` localiza una función o método por nombre usando `go/ast` (no matching de texto) y reemplaza toda su declaración, dejando el resultado formateado con `go/format` y verificado sintácticamente antes de escribir. Es la herramienta preferida para cambios de lógica/firma de una función existente.
- `replaceExact` sirve para ediciones puntuales (una línea de import, un call site) donde un snippet corto y único es suficiente.
- `createFile` / `deleteFile` cubren archivos nuevos o eliminación de archivos.

El prompt del *Code Builder* recibe la tarea, el plan, los findings del Analyzer, la lista de archivos del workspace **y el contenido de esos archivos**, con instrucciones explícitas para hacer cambios mínimos y dirigidos en vez de reescribir archivos completos.

Esto le da al agente el poder expresivo de código real (loops, condicionales, composición de varios cambios en una sola Action) sin dejar de estar acotado a un conjunto reducido y auditable de operaciones sobre el sistema de archivos.

---

## Arquitectura y flujo

```
Repository
    ↓
Analyzer (go/ast) → Findings
    ↓
Agent (LLM: plan → code)
    ↓
Action (Go generado)
    ↓
Executor (build + run en workspace aislado)
    ↓
Validator (go test ./...)
    ↓
Comparator (diff original vs workspace)
    ↓
Human Approval
    ↓
Apply (solo si se aprueba)
```

Componentes principales (`internal/`):

| Paquete      | Responsabilidad                                                        |
|--------------|--------------------------------------------------------------------------|
| `workspace`  | Copia aislada del proyecto; lectura/escritura restringida a su raíz.    |
| `analyzer`   | Detecta findings (`function_too_long`, `too_many_parameters`) vía AST.  |
| `prompt`     | Construye los prompts (plan, código, feedback correctivo).             |
| `llm`        | Cliente contra Gemini.                                                 |
| `action`     | Envoltorio del código generado (limpia code fences, etc.).             |
| `executor`   | Escribe, compila y ejecuta la Action junto a `refactai_tools.go`.       |
| `validator`  | Corre `go test ./...` sobre el workspace si hay `go.mod`.              |
| `agent`      | Orquesta el loop: plan → code → run → validate → retry (máx. 3).       |
| `comparator` | Genera el diff y aplica los cambios aprobados al proyecto original.    |
| `config`     | Carga configuración (API key, modelo) desde variables de entorno.      |
| `cmd/agent`  | CLI: entrada, interacción con el usuario, aprobación de cambios.       |

---

## Aislamiento y seguridad

- El agente **nunca** trabaja directamente sobre el proyecto original: todo ocurre sobre una copia temporal (`workspace`).
- `workspace.ResolvePath` impide rutas absolutas y cualquier intento de escapar de la raíz del workspace (`../`).
- La Action generada por el LLM se ejecuta como binario compilado, con `cmd.Dir` fijado a la raíz del workspace.
- Los cambios solo llegan al proyecto real a través de `Comparator.Apply`, y solo después de que el usuario los revisa y aprueba explícitamente en el diff mostrado por CLI.
- Si el usuario rechaza, el workspace temporal se elimina (`defer os.RemoveAll(...)`) y el proyecto original no se toca.

---

## Requisitos

- Go 1.23 o superior.
- Una API key de Gemini (variable `GEMINI_API_KEY`).

---

## Instalación

```bash
git clone https://github.com/AndresAlcantaraM/RefactAI.git
cd RefactAI
go mod download
```

---

## Configuración

Crear un archivo `.env` en la raíz del proyecto (se carga automáticamente vía `godotenv`):

```
GEMINI_API_KEY=tu_api_key_aqui
```

---

## Uso

```bash
go run ./cmd/agent <project-path> "<task>"
```

- `<project-path>`: ruta al proyecto Go que se quiere refactorizar.
- `<task>`: descripción en lenguaje natural de lo que se quiere lograr.

El CLI muestra el plan generado, el código de la Action, el resultado de ejecución/validación y, si hay cambios, el diff completo antes de pedir confirmación:

```
Apply these changes to the project? [y/N]:
```

---

## Ejemplo

```bash
go run ./cmd/agent ./demo "Refactor the code to address the analyzer findings while preserving existing behavior."
```

El agente detecta findings como `too_many_parameters` en `notification.go`, genera un plan (por ejemplo, agrupar parámetros en un struct), produce una Action que usa `replaceFunction` para reescribir solo esas funciones, valida que el proyecto siga compilando/pasando tests, y muestra el diff resultante para aprobación.

---

## Tests

```bash
go test ./...
```

Incluye tests unitarios por paquete (`analyzer`, `executor`, `comparator`, `validator`, `workspace`, `agent`) y un test **end-to-end** (`internal/e2e`) que ejercita el flujo completo: workspace aislado → agent → comparator → diff → apply → verificación del proyecto original modificado.

---

## Estado del proyecto

Este es un **PoC (proof of concept) funcional**, no un producto terminado. El flujo completo (analyze → plan → code → execute → validate → diff → approve → apply) funciona de punta a punta para proyectos Go de un solo paquete.

**Completado:**

- Fundación: workspace, analyzer, prompt builder, integración LLM, action, executor, validator, agent loop con reintentos.
- Seguridad del flujo: workspace aislado, ejecución restringida, validación, diff, aprobación humana, apply/discard.
- CLI mínimo funcional: ejecución, plan, resultado, diff, aprobación.
- Herramienta de edición determinista basada en AST (`replaceFunction`), que reemplazó el enfoque inicial de reconstrucción de archivos por texto, mucho más propenso a error.

---

## Qué falta / próximos pasos

Estas son mejoras conocidas y deliberadamente pospuestas para no bloquear la entrega del PoC — no afectan el funcionamiento del caso principal (proyecto Go de un solo paquete):

- **Multi-paquete en un mismo directorio**: si el proyecto de prueba mezcla archivos de distintos `package` en una carpeta plana, `go test ./...` falla con `found packages X and Y`. No es un bug del agente sino de cómo está organizado ese escenario de prueba; se resuelve separando por subcarpetas o ajustando el validator para no asumir un único paquete por directorio.
- **Desambiguación de `replaceFunction`** cuando existen dos funciones/métodos con el mismo nombre (distinto receiver) en el mismo archivo.
- **Findings más estructurados**: hoy el nombre de la función afectada va embebido en `Finding.Message`; sería más robusto tener un campo explícito `Function string`.
- Tests de integración end-to-end adicionales, específicamente para `replaceFunction`.
- Manejo más robusto de cancelación de contexto y casos límite de paths.
- Limpieza garantizada del workspace ante fallos inesperados.
- Mejor UX de CLI (manejo de errores, salida más legible).
- Observabilidad/logging, configuración externa, sandboxing real de la ejecución de la Action.

---

## Decisiones de diseño

- **Determinismo sobre flexibilidad**: el agente está deliberadamente limitado a actuar sobre lo que reporta el Analyzer, en vez de interpretar cualquier tarea como licencia para rediseñar el proyecto libremente.
- **AST sobre matching de texto**: la primera versión de la herramienta de edición (`replaceExact`) obligaba al LLM a reproducir código existente carácter por carácter dentro de un string, lo cual fallaba con frecuencia en funciones más complejas. `replaceFunction` resuelve esto localizando la función por AST y delegando el formateo/validación sintáctica a las herramientas estándar de Go (`go/parser`, `go/format`), en vez de confiar en que el modelo transcriba texto exacto.
- **Human-in-the-loop obligatorio**: ningún cambio llega al proyecto real sin aprobación explícita del usuario sobre un diff visible.
