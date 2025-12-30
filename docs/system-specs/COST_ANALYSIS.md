# Cost Analysis & Token Usage

| Metric                          | Current Value           |
| ------------------------------- | ----------------------- |
| **Cost per request**            | ~$0.032 (~3 cents)      |
| **Total tokens per request**    | ~10,100 tokens          |
| **Input tokens (Sonnet)**       | ~10,000 tokens          |
| **Output tokens (Sonnet)**      | ~100 tokens             |
| **Cost at 1,000 requests**      | $32                     |
| **Cost at 10,000 requests/day** | $320/day = $9,600/month |

**Verdict:** Current cost is acceptable for quality provided.

---

## Token Breakdown

```
User Query → Query Transform (Haiku) → Hybrid Search → System Prompt → Generate (Sonnet)
```

### Input Tokens (~10,000)

| Component                 | Tokens | % of Total |
| ------------------------- | ------ | ---------- |
| Cheatsheet                | 3,600  | 36%        |
| Instructions              | 2,000  | 20%        |
| Documentation (3 chunks)  | 2,400  | 24%        |
| Examples (2 patterns)     | 1,000  | 10%        |
| Editor State + History    | 800    | 8%         |
| User Query                | 200    | 2%         |
| **TOTAL**                 | 10,000 | 100%       |

### Output Tokens (~100 average)

Typical response: 50-150 tokens (code snippet with brief explanation)

---

## Cost Calculations

### Pricing (Claude Sonnet 4)

- Input: $3.00 / 1M tokens
- Output: $15.00 / 1M tokens

### Per Request

```
Input:  10,000 tokens × $3.00/1M  = $0.030
Output:    100 tokens × $15.00/1M = $0.0015
Haiku overhead:                     $0.0001
──────────────────────────────────────────
TOTAL:                              ~$0.032
```

**Key insight:** Cheatsheet + Instructions = 56% of input tokens

---

## Scale Projections

| Requests/Day | Cost/Day | Cost/Month |
| ------------ | -------- | ---------- |
| 100          | $3.20    | $96        |
| 1,000        | $32      | $960       |
| 10,000       | $320     | $9,600     |
| 100,000      | $3,200   | $96,000    |

---

## Optimization Options

**If costs exceed $200/day:**
- Reduce doc chunks: 3 → 2 (-800 tokens, saves ~$0.0024/request)
- Reduce examples: 2 → 1 (-500 tokens, saves ~$0.0015/request)

**If costs exceed $500/day:**
- Compress cheatsheet to essentials (-1,000 tokens)
- Limit conversation history (-300 tokens)

**Emergency only:**
- Aggressive trimming impacts quality significantly - not recommended

---

## When to Optimize

- **< $200/day**: No optimization needed
- **$200-500/day**: Consider light optimizations
- **> $500/day**: Implement optimizations, A/B test quality impact

## Measuring Tokens

Check Claude API response headers:
```
x-anthropic-input-tokens: 10000
x-anthropic-output-tokens: 100
```
