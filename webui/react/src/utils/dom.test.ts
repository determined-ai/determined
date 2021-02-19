import { ansiToHtml, toPixel, toRem } from './dom';

describe('ansiToHtml', () => {
  it('should convert ANSI colors', () => {
    expect(ansiToHtml('\u001b[30m30\u001b[0m')).toBe('<span style="color:#000">30</span>');
    expect(ansiToHtml('\u001b[31m31\u001b[0m')).toBe('<span style="color:#A00">31</span>');
    expect(ansiToHtml('\u001b[32m32\u001b[0m')).toBe('<span style="color:#0A0">32</span>');
    expect(ansiToHtml('\u001b[33m33\u001b[0m')).toBe('<span style="color:#A50">33</span>');
    expect(ansiToHtml('\u001b[34m34\u001b[0m')).toBe('<span style="color:#00A">34</span>');
    expect(ansiToHtml('\u001b[35m35\u001b[0m')).toBe('<span style="color:#A0A">35</span>');
    expect(ansiToHtml('\u001b[36m36\u001b[0m')).toBe('<span style="color:#0AA">36</span>');
  });
});

describe('toRem', () => {
  it('should convert number to rem value', () => {
    expect(toRem(5)).toBe('0.5rem');
    expect(toRem(123)).toBe('12.3rem');
    expect(toRem(0)).toBe('0rem');
  });

  it('should convert pixel value to rem value', () => {
    expect(toRem('5px')).toBe('0.5rem');
    expect(toRem('12 px')).toBe('1.2rem');
    expect(toRem('50.7px')).toBe('5.07rem');
  });

  it('should leave rem values alone', () => {
    expect(toRem('0.5rem')).toBe('0.5rem');
    expect(toRem('10 rem')).toBe('10rem');
    expect(toRem('123.45 rem')).toBe('123.45rem');
  });
});

describe('toPixel', () => {
  it('should convert number to pixel value', () => {
    expect(toPixel(0)).toBe('0px');
    expect(toPixel(5)).toBe('50px');
    expect(toPixel(5.5)).toBe('55px');
    expect(toPixel(123)).toBe('1230px');
  });

  it('should convert rem value to pixel value', () => {
    expect(toPixel('.5rem')).toBe('5px');
    expect(toPixel('0.5rem')).toBe('5px');
    expect(toPixel('5rem')).toBe('50px');
    expect(toPixel('12 rem')).toBe('120px');
    expect(toPixel('50.7rem')).toBe('507px');
  });

  it('should leave px values alone', () => {
    expect(toPixel('0.5px')).toBe('0.5px');
    expect(toPixel('10 px')).toBe('10px');
    expect(toPixel('123.45 px')).toBe('123.45px');
  });
});
