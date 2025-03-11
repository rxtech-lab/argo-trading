# Commission Formula Documentation

The Argo Trading backtesting engine supports dynamic commission calculation using either Go templates or human-readable mathematical expressions powered by the [expr-lang/expr](https://github.com/expr-lang/expr) library. This allows you to define complex commission structures based on order properties.

## Configuration

In your `config.yaml` file, use the `commission_formula` field to define how commissions should be calculated:

### Using Human-Readable Expressions (Recommended)

```yaml
initial_capital: 10000
start_time: 2025-01-01T00:00:00Z
end_time: 2025-02-02T23:59:59Z
results_folder: results
commission_formula: "0.001 * total + 0.02" # 0.1% of the total order value plus $0.02 fixed fee
```

### Using Go Templates (Advanced)

```yaml
initial_capital: 10000
start_time: 2025-01-01T00:00:00Z
end_time: 2025-02-02T23:59:59Z
results_folder: results
commission_formula: "{{ mul .Total 0.001 }}" # 0.1% of the total order value
```

## Human-Readable Expressions (expr-lang/expr)

Human-readable expressions are easier to write and understand. They support standard mathematical operators, functions, and conditional logic.

### Available Variables

| Variable    | Description                                      |
| ----------- | ------------------------------------------------ |
| `quantity`  | The quantity of the order                        |
| `price`     | The execution price of the order                 |
| `total`     | The total value of the order (quantity \* price) |
| `symbol`    | The symbol being traded                          |
| `orderType` | The type of order ("BUY" or "SELL")              |

### Available Operators

| Operator | Description           | Example                                |
| -------- | --------------------- | -------------------------------------- |
| `+`      | Addition              | `0.001 * total + 5`                    |
| `-`      | Subtraction           | `total - 100`                          |
| `*`      | Multiplication        | `0.001 * total`                        |
| `/`      | Division              | `total / 100`                          |
| `%`      | Modulo                | `total % 100`                          |
| `==`     | Equal                 | `orderType == "BUY"`                   |
| `!=`     | Not equal             | `orderType != "SELL"`                  |
| `>`      | Greater than          | `total > 1000`                         |
| `>=`     | Greater than or equal | `total >= 1000`                        |
| `<`      | Less than             | `total < 1000`                         |
| `<=`     | Less than or equal    | `total <= 1000`                        |
| `&&`     | Logical AND           | `total > 1000 && orderType == "BUY"`   |
| `\|\|`   | Logical OR            | `total < 500 \|\| orderType == "SELL"` |
| `!`      | Logical NOT           | `!(orderType == "BUY")`                |
| `?:`     | Ternary conditional   | `total > 1000 ? 0.001 : 0.002`         |

### Available Functions

| Function | Description               | Example                    |
| -------- | ------------------------- | -------------------------- |
| `max`    | Maximum of two values     | `max(1.0, 0.001 * total)`  |
| `min`    | Minimum of two values     | `min(10.0, 0.002 * total)` |
| `abs`    | Absolute value            | `abs(total - 1000)`        |
| `sqrt`   | Square root               | `sqrt(total)`              |
| `pow`    | Power                     | `pow(total, 0.5)`          |
| `ceil`   | Round up to nearest int   | `ceil(total * 0.001)`      |
| `floor`  | Round down to nearest int | `floor(total * 0.001)`     |
| `round`  | Round to nearest int      | `round(total * 0.001)`     |

### Example Human-Readable Formulas

#### Percentage-based Commission

```yaml
commission_formula: "0.001 * total" # 0.1% of the total order value
```

#### Fixed Fee Plus Percentage

```yaml
commission_formula: "0.0005 * total + 5" # 0.05% of the total order value plus $5 fixed fee
```

#### Tiered Commission Structure

```yaml
commission_formula: "total < 1000 ? 0.002 * total : 0.001 * total"
```

This charges 0.2% for orders under $1000 and 0.1% for orders $1000 and above.

#### Minimum Commission

```yaml
commission_formula: "max(1.0, 0.001 * total)"
```

This charges 0.1% of the total order value with a minimum commission of $1.00.

#### Different Fees for Buy and Sell Orders

```yaml
commission_formula: 'orderType == "BUY" ? 0.001 * total : 0.002 * total'
```

This charges 0.1% for buy orders and 0.2% for sell orders.

#### Symbol-specific Fees

```yaml
commission_formula: 'symbol == "AAPL" ? 0.0005 * total : 0.001 * total'
```

This charges 0.05% for AAPL trades and 0.1% for all other symbols.

#### Complex Tiered Structure

```yaml
commission_formula: "total < 500 ? 0.003 * total : (total < 1000 ? 0.002 * total : 0.001 * total)"
```

This charges:

- 0.3% for orders under $500
- 0.2% for orders between $500 and $1000
- 0.1% for orders $1000 and above

#### Fixed Fee with Minimum

```yaml
commission_formula: "max(1.0, 0.001 * total + 0.5)"
```

This charges a $0.50 fixed fee plus 0.1% of the total order value, with a minimum commission of $1.00.

#### Volume Discount

```yaml
commission_formula: "quantity > 100 ? 0.0008 * total : 0.001 * total"
```

This charges 0.08% for orders with quantity greater than 100, and 0.1% for smaller orders.

#### Combined Conditions

```yaml
commission_formula: '(orderType == "BUY" && total > 1000) ? 0.0008 * total : 0.001 * total'
```

This charges 0.08% for buy orders over $1000, and 0.1% for all other orders.

## Go Templates (Advanced)

Go templates provide more advanced functionality but are more complex to write.

### Available Variables

| Variable     | Description                                      |
| ------------ | ------------------------------------------------ |
| `.Quantity`  | The quantity of the order                        |
| `.Price`     | The execution price of the order                 |
| `.Total`     | The total value of the order (Quantity \* Price) |
| `.Symbol`    | The symbol being traded                          |
| `.OrderType` | The type of order ("BUY" or "SELL")              |

### Available Functions

| Function | Description    | Example                      |
| -------- | -------------- | ---------------------------- |
| `mul`    | Multiplication | `{{ mul .Total 0.001 }}`     |
| `div`    | Division       | `{{ div .Total 100 }}`       |
| `add`    | Addition       | `{{ add .Total 5 }}`         |
| `sub`    | Subtraction    | `{{ sub .Total 5 }}`         |
| `min`    | Minimum        | `{{ min .Total 10 }}`        |
| `max`    | Maximum        | `{{ max .Total 10 }}`        |
| `abs`    | Absolute value | `{{ abs (sub .Total 100) }}` |

### Example Go Template Formulas

#### Percentage-based Commission

```yaml
commission_formula: "{{ mul .Total 0.001 }}" # 0.1% of the total order value
```

#### Tiered Commission Structure

```yaml
commission_formula: "{{ if lt .Total 1000 }}{{ mul .Total 0.002 }}{{ else }}{{ mul .Total 0.001 }}{{ end }}"
```

This charges 0.2% for orders under $1000 and 0.1% for orders $1000 and above.

#### Fixed Fee Plus Percentage

```yaml
commission_formula: "{{ add 5 (mul .Total 0.0005) }}" # $5 + 0.05% of the total order value
```

#### Different Fees for Buy and Sell Orders

```yaml
commission_formula: '{{ if eq .OrderType "BUY" }}{{ mul .Total 0.001 }}{{ else }}{{ mul .Total 0.002 }}{{ end }}'
```

This charges 0.1% for buy orders and 0.2% for sell orders.

#### Symbol-specific Fees

```yaml
commission_formula: '{{ if eq .Symbol "AAPL" }}{{ mul .Total 0.0005 }}{{ else }}{{ mul .Total 0.001 }}{{ end }}'
```

This charges 0.05% for AAPL trades and 0.1% for all other symbols.

#### Advanced Usage

```yaml
commission_formula: '{{ max 1.0 (mul .Total (if eq .OrderType "BUY" 0.001 0.002)) }}'
```

This charges 0.1% for buy orders and 0.2% for sell orders, with a minimum commission of $1.00.
