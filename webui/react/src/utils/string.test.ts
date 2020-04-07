import { toHtmlId } from './string';

describe('toHtmlId', () => {
  it('should replace spaces with -', () => {
    expect(toHtmlId('Hello World')).toBe('hello-world');
  });

  it('should remove everything but alphanumeric and -', () => {
    expect(toHtmlId('He$)%*#$%llo World)#$(%)')).toBe('hello-world');
  });

  it('should generate lowercase ids', () => {
    expect(toHtmlId('HellO')).toBe('hello');
  });

});
