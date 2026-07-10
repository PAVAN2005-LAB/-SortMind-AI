def factorial(n):
    """
    Calculate the factorial of a non-negative integer n.
    """
    if n == 0 or n == 1:
        return 1
    return n * factorial(n - 1)

if __name__ == "__main__":
    number = 5
    result = factorial(number)
    print(f"The factorial of {number} is {result}")
