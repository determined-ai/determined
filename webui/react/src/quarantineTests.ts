export const quarantinedIt = (name: string, fn?: jest.ProvidesCallback, timeout?: number): void => {
  return (process.env.QUARANTINED ? it : it.skip)(name, fn, timeout);
};
