import { defineConfig } from 'orval';

export default defineConfig({
  e2e: {
    input: '../../proto/build/swagger/determined/api/v1/api.swagger.json',
    output: {
      target: './generated/orval/e2e-client.ts',
    },
  },
});
