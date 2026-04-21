export const jqlExamples = [
  `status = "active"`,
  `name contains "产品"`,
  `price > 100`,
  `status IN ("active", "pending")`,
  `category NOT IN ("deleted", "archived")`,
  `title LIKE "重要%"`,
  `created > "2024-01-01"`,
  `assignee IS NULL`,
  `email IS NOT NULL`,
  `status = "active" AND price > 100`,
  `name = "A" OR name = "B"`,
  `(status = "active") AND (price > 100 OR price < 50)`,
  `created > StartOfWeek() AND module = "movie"`,
  `updated < EndOfMonth() AND status NOT IN ("deleted", "archived")`,
]

export const JQL_OPERATORS = {
  comparision: ['=', '!=', '>', '<', '>=', '<=', '~'],
  logical: ['AND', 'OR', 'NOT'],
  special: ['IN', 'NOT IN', 'IS NULL', 'IS NOT NULL', 'LIKE', 'contains'],
}

export const JQL_FUNCTIONS = [
  'currentUser()',
  'now()',
  'startOfDay()',
  'endOfDay()',
  'startOfWeek()',
  'endOfWeek()',
  'startOfMonth()',
  'endOfMonth()',
]

export const parseJQL = (query: string): { valid: boolean; error?: string } => {
  if (!query || query.trim() === '') {
    return { valid: true }
  }

  const keywords = ['AND', 'OR', 'NOT', 'IN', 'IS', 'NULL', 'NOT', 'LIKE', 'contains', 'startOfDay', 'endOfDay', 'startOfWeek', 'endOfWeek', 'startOfMonth', 'endOfMonth', 'currentUser', 'now']

  for (const keyword of keywords) {
    const regex = new RegExp(`\\b${keyword}\\b`, 'gi')
    const matches = query.match(regex)
    if (matches) {
      return { valid: true }
    }
  }

  return { valid: true }
}