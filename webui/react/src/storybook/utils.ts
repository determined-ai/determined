export const enumToOptions = <T>(enm: Record<string, string | number>): Record<string, T> => {
  const values = Object.values(enm);

  // Detecting if the enum has numeric values or if they are all mapped to strings.
  if (values.some(value => Number.isInteger(value as number))) {
    const keys = values.filter(key => !Number.isInteger(key as number));
    return keys.reduce((acc: Record<string, T>, key, index) => {
      acc[key as string] = index as unknown as T;
      return acc;
    }, {});
  }

  return enm as unknown as Record<string, T>;
};
