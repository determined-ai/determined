# Page Model Guide

Models represented in this folder are meant to be a one-to-one representations of objects from `src`.

- `src/pages/SignIn.tsx` -> `src/e2e/models/pages/SignIn.ts`
- `src/components/DeterminedAuth.tsx` -> `src/e2e/models/components/DeterminedAuth.ts`
- `src/utils/error.ts` -> `src/e2e/models/utils/error.ts`

## Why page models

Playwright has [a writeup](https://playwright.dev/docs/pom) on how page models are useful, namely that they can be used as an API for interacting with pages. While this is true, there is much more to be gained from using page models.

### What playwright POM is missing

The playwright API approach still loosely defines each component. Relying on text can be volitile, so it would be better to use accessibility or `data-testid`. Playwright POM also serves as an API rather than a **model**.

Our page models are meant to implement the precise structure of a page. This allows us to be specific in which components we represent. For exmaple, a login page might have different sign in boxes for straightforward authetication and for SSO. Or maybe a dashboard page has several tables. When we test for "a username input" or "the first row of the table", these models will provide a uniform, specific mechanism to clearly state which component we're using.

In addition, playwright POM muddies their implementation with test steps executing inside the model. We want our models to represent a page that a test can perform actions on instead of acting as an interface to perform actions on that page. Here's an example.

| Unclear Test Case | `jspage.createNewProject()`                            |
| ----------------- | ------------------------------------------------------ |
| Precise Test Case | `await page.header.createNewProject.pwLocator.click()` |

The difference is suble, but in the precise exmaple, a reader will understand exactly what actions the test case is performing from a glance. This can be helpful when test cases fail and the person triaging the case isn't the same person who wrote the test. Utilities and abstarctions are still welcome, but this model pattern will elimate the need for as many!

## How to use page models

Simply instantiate a new page inside a test case or fixture. The page will come with properties representing the page. The `pwLocator` property gets the playwright locator which the model is representing.

| Page model Autocomplete                                                 | Locator automcomplete                                             |
| ----------------------------------------------------------------------- | ----------------------------------------------------------------- |
| ![page model automcomplete](../docs/images/page-model-autocomplete.png) | ![locator automcomplete](../docs/images/loactor-autocomplete.png) |

And here's a complete example with an `expect`:

```js
await detAuth.username.pwLocator.fill(username);
await detAuth.password.pwLocator.fill(password);
expect(await signInPage.detAuth.error.message.pwLocator.textContent()).toContain('Login failed');
```

## How to contribute

First, if the model doesn't exist, create it. The file name and path in `src/e2e/models` with match the file name and path from it's counterpart in `src`.

- Pages will inherit from `BasePage` and serve as the root of a component tree. Each page will need a `url`, and the `BasePage` class provides common properties for all pages like `goto`.
- Named components and utilities will inherit form `NamedComponent`. This class is similar to `BaseComponent` but it enforces that a static `defaultSelector` be declared. Use this selector in the component's constructor as an alternative if `selector` isn't passed through.
- If the component being represented returns a `React.Fragment`, inherit from the `BaseReactFragment` class

All pages and components support subcomponents using instance properties. Initalize new components using instances of BaseComponent. They should support selector and parent arguments.

| Argument   | Description                                                                                                                                                                                                                            |
| ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `selector` | The selector passed through to playwright's `locator` method. Required if initalizing a `BaseComponent`, but optional for `NamedComponent`s.                                                                                           |
| `parent`   | The component's parent. It could be `this` or any instance of `BaseComponent \| BasePage`. In some cases, it will be set to `this.root` if the component lives at the root of the DOM. `BasePage`s are always at the root of the tree. |

Here's a simplified example using part of the `DeterminedAuth` component. Intrinsic elements are be represent with `BaseComponent`, and they can be as `parent`s for other elements. The amount of specificity is left to the author's discretion. If you're not sure, more is better to avoid conflicts with future additions. Deeper specificity will also optimize for searching through the DOM on larger pages.

```js
export class DeterminedAuth extends NamedComponent({ defaultSelector: "div[data-test-component='detAuth']"}) {
  constructor({ selector, parent }: NamedComponentArgs) {
    super({ parent: parent, selector: selector || DeterminedAuth.defaultSelector });
  }
  readonly #form: BaseComponent = new BaseComponent({ parent: this, selector: 'form' });
  readonly username: BaseComponent = new BaseComponent({
    parent: this.#form,
    selector: "input[data-testid='username']",
  });
  readonly password: BaseComponent = new BaseComponent({
    parent: this.#form,
    selector: "input[data-testid='password']",
  });
  readonly submit: BaseComponent = new BaseComponent({
    parent: this.#form,
    selector: "button[data-testid='submit']",
  });
  ...
}
```

### Default Selectors

`NamedComponent`s will have a `static defaultSelector` to use if a selector isn't provided. Since this property is static, we can use it to create elements with more details. Here's an example using an imaginary `DeterminedTable` component:

```js
  readonly userTable: DeterminedTable = new DeterminedTable({ parent: this, selector: DeterminedTable.defaultSelector + "[data-testid='userTable']" });
  readonly roleTable: DeterminedTable = new DeterminedTable({ parent: this, selector: DeterminedTable.defaultSelector + "[data-testid='roleTable']" });
```

## Practices around test hooks

When creating page models, you'll most likely want to author test hooks into the `src` model.

| Test Hook                            | Usage                                                                |
| ------------------------------------ | -------------------------------------------------------------------- |
| `data-test-component='my-component'` | Belongs at the top level element wrapping the component              |
| `data-testid='my-componentid'`       | Attributed to any _instances_ of components or any intrinsic element |

Looking back to the exmaple with the imaginary `DeterminedTable`, we want to enable this pattern:

```js
  // DeterminedTable.defaultSelector = `[data-test-component='DetTable']`
  readonly userTable: DeterminedTable = new DeterminedTable({ parent: this, selector: DeterminedTable.defaultSelector + "[data-testid='userTable']" });
  readonly roleTable: DeterminedTable = new DeterminedTable({ parent: this, selector: DeterminedTable.defaultSelector + "[data-testid='roleTable']" });
```

The component `DeterminedTable` would have `data-test-component='DetTable'` as a top level attribute, and instances would each get their own `data-testid`. This way, the static attribute and the instance attribute don't conflict with each other.

Not every component needs a data-testid, but, in general, more is better. It's better to select for _"a duck named Hoffman"_ rather than "a duck" or "Hoffman".
