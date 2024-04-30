import { test } from '@playwright/test';

import { makeCommand } from './mock.utils';

test.describe('Save recorded Mocks', async () => {
    await makeCommand('mb-save-imposters');
});