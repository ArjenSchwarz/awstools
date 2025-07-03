1. Read the requirements.md file in plans/$ARGUMENTS/
2. Search for and go through the code referenced in the requirements file
3. Write a design.md file in plans/$ARGUMENTS/ following the below template

```
# [Feature Name] Design Document

## Overview

[Provide a brief introduction to the feature and its purpose. Explain the problem it solves and the benefits it provides to users. This section should be 2-3 paragraphs that give readers a clear understanding of what the feature is and why it's valuable.]

This feature will [describe the primary benefit or outcome for users].

## Architecture

[Describe how the feature fits into the existing architecture of the application. Include a high-level description of the components involved and how they interact.]

The [Feature Name] will integrate with the existing workflow:

1. [Step 1 of integration]
2. [Step 2 of integration]
3. [Step 3 of integration]
4. [Step 4 of integration]

## Components and Interfaces

[Break down the feature into its component parts and describe the interfaces between them. Include code snippets where appropriate to illustrate the design.]

### [Component 1]

[Describe the first component and its responsibilities]


    // Example code snippet showing the interface or implementation
    type ExampleComponent struct {
        // Fields
    }

    func (e *ExampleComponent) ExampleMethod() {
        // Implementation details
    }


### [Component 2]

[Describe the second component and its responsibilities]


    // Example code snippet
    func ExampleFunction() {
        // Implementation details
    }


### [Component 3]

[Describe the third component and its responsibilities]

## Data Models

[Describe any data models or structures that will be created or modified for this feature.]


    type ExampleModel struct {
        Field1 string `json:"field1"`
        Field2 int    `json:"field2"`
        // Additional fields
    }


## Error Handling

[Describe how errors will be handled in the feature implementation.]

1. [Error handling approach 1]
2. [Error handling approach 2]
3. [Error handling approach 3]

## Testing Strategy

[Outline the testing strategy for the feature, including unit tests, integration tests, and any manual testing required.]

We'll implement the following tests:

1. [Test type 1]
2. [Test type 2]
3. [Test type 3]
4. [Test type 4]

Test cases will include:

- [Test case 1]
- [Test case 2]
- [Test case 3]
- [Test case 4]

## Implementation Plan

[Provide a step-by-step plan for implementing the feature.]

1. [Implementation step 1]
2. [Implementation step 2]
3. [Implementation step 3]
4. [Implementation step 4]
5. [Implementation step 5]

## Conclusion

[Summarize the design and highlight any important considerations or trade-offs.]

This design for [Feature Name] provides [key benefit 1] and [key benefit 2] while maintaining [important constraint or compatibility]. The implementation approach prioritizes [key priority] and ensures [important quality attribute].
```