#!/bin/bash

LINES_WITH_ASSERT=$(git ls-files -z '*.py' ':!:*tests/*' ':!:dist/*' ':!:examples/*' | xargs -0 grep -n -H "assert ")
if [ -n "$LINES_WITH_ASSERT" ]; then
  echo "$LINES_WITH_ASSERT"
  echo "We no longer support \`assert\`."
  echo "Consider using \`typing.cast\` to please Mypy or using our \`check\` package if you intend to error."
  exit 1
fi
