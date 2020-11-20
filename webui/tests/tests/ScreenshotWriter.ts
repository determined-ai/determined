import { CustomScreenshotWriter } from 'gauge-ts';
import { join, basename } from 'path';

const { screenshot } = require('taiko');

export default class ScreenshotWriter {
  @CustomScreenshotWriter()
  public async foo(): Promise<string> {
    const screenshotFilePath = join(
      process.env['gauge_screenshots_dir'],
      `screenshot-${process.hrtime.bigint()}.png`,
    );
    await screenshot({ path: screenshotFilePath });
    return basename(screenshotFilePath);
  }
}
