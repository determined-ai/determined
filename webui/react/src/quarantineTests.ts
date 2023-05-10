export const flakyIt = (name: string, fn?: jest.ProvidesCallback, timeout?: number): void => {
  return (process.env.INCLUDE_FLAKY ? it : it.skip)(name, fn, timeout);
};
