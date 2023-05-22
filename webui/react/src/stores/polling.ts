import handleError from 'utils/error';

const POLLING_DELAY_MS = 10_000;
const RETRY_DELAY_MS = 1000;
const MAX_RETRY_DELAY_MS = 600_000;

// Less than 0 means to keep trying indefinitely.
const MAX_RETRY = -1;
const EMPTY_FUNCTION = () => {
  return;
};

interface PollingOptions {
  args?: unknown[];
  condition?: boolean;
  delay?: number;
  maxRetry?: number;
}

/**
 * To use this polling store...
 *
 * 1. Take your store and extend this `PollingStore` class.
 *
 *    class MyStore extends PollingStore { ... }
 *
 * 2. Then you simply need to override the `poll` function.
 *    This is executed during each polling iteration.
 *    You can also override the `pollCatch` function, but it is completely optional.
 *    `pollCatch` is executed when an error is encountered during the `poll`
 *    such as network issues or API failures.
 *
 * 3. Call the `startPolling` and `stopPolling` functions to
 *    start and stop the polling. The most common way is to do this
 *    in a `React.useEffect` as such:
 *
 *    useEffect(() => myStore.startPolling(), []);
 *
 *    `startPolling` returns `stopPolling` appropriately so during a
 *    React unmount it will call the `stopPolling` routine automatically.
 */
abstract class PollingStore {
  protected canceler?: AbortController;
  protected pollingTimer?: NodeJS.Timer;
  protected pollingArgs: unknown[] = [];
  protected pollingDelay = POLLING_DELAY_MS;
  protected pollingRetry = 0;
  protected maxRetry = MAX_RETRY;

  // This must be overriden if `PollingStore` is extended.
  protected abstract poll(...args: unknown[]): Promise<unknown>;

  // Overriding this is optional.
  protected pollCatch(): void {
    return;
  }

  /**
   * Polling behavior of starting the delay timer upon a success or failed response.
   * Upon failure, polling will retry but with longer and longer retry delays.
   * Upon a successful response after a failure, the retry count resets so the
   * retry delay becomes short again.
   */
  protected async pollFn(...args: unknown[]): Promise<void> {
    try {
      await this.poll(...args);
      this.pollingRetry = 0;
      this.pollingTimer = setTimeout(() => this.pollFn(...args), this.pollingDelay);
    } catch (e) {
      this.pollCatch();
      if (this.maxRetry < 0 || this.pollingRetry < this.maxRetry) {
        this.pollingTimer = setTimeout(
          () => this.pollFn(...args),
          Math.min(MAX_RETRY_DELAY_MS, RETRY_DELAY_MS * Math.pow(2, this.pollingRetry)),
        );
        this.pollingRetry++;
      }
      handleError(e);
    }
  }

  public startPolling(options: PollingOptions = {}): () => void {
    if (!(options.condition ?? true)) return EMPTY_FUNCTION;

    this.pollingArgs = options.args ?? [];
    this.pollingDelay = options.delay ?? POLLING_DELAY_MS;
    this.maxRetry = options.maxRetry ?? MAX_RETRY;
    this.stopPolling();

    this.canceler = new AbortController();
    this.pollFn(...this.pollingArgs);

    /**
     * `stopPolling` returned for `useEffect` convenience.
     * `.bind(this)` is required to preserve the context of the store.
     */
    return this.stopPolling.bind(this);
  }

  public stopPolling(): void {
    this.canceler?.abort();

    if (this.pollingTimer) {
      clearTimeout(this.pollingTimer);
      this.pollingTimer = undefined;
    }
  }
}

export default PollingStore;
