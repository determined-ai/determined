import { test } from '@playwright/test';

import {
  makeCommand,
  wait,
} from './mock.utils';

test.describe('Start Mocks', async () => {
    await makeCommand('mb-stop');
    await makeCommand('mb-start');
    let success = false;
    for (let count = 0; count < 5 && !success; count++) {
        success = await makeCommand('mb-record-imposters');
        wait(count * 500);
    }
});



