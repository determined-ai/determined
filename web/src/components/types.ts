export interface ConditionalButton<T> {
  button: React.ReactNode;
  showIf?: (item: T) => boolean;
}
