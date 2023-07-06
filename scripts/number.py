# Define the file name
file_name = "numbers.txt"

# Open the file in write mode
with open(file_name, "w") as file:
    # Write numbers 1 to 100 in the file
    for number in range(1, 101):
        file.write(str(number) + "\n")

print("Data written to the file:", file_name)
