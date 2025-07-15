import json
import subprocess
import re
import tkinter as tk
from tkinter import ttk
from tkinter import messagebox

# Global Variables
active_editor = []
main_tree = {}
worker_config = {}
ui_config = {}
root = {}
columns = {}
rows = {}
tickers = [
    "BTC", "ETH", "USDT", 
    "SOL", "SUI", "PUMP", "VIRTUAL", "KAIA", 
    "IP", "S", "XRP", "TRX", "DOGE", 
    "ADA", "HYPE", "LINK", "AVAX", 
    "SHIB", "HBAR", "VET"
]
comparison = ["<", ">", "="]


# Destroy active editor
def destroy_active_editor():
    global active_editor, main_tree

    if len(active_editor) > 0:
        active_editor[0].event_generate("<FocusOut>")
        # active_editor[0].event_generate("<Return>")
        active_editor[0].destroy()

        active_editor = []


# Special callback to close the active editor when user clicked outside the elements
def click_outside_item_callback(event):
    global active_editor, main_tree
    if event:
        item = main_tree.identify('item', event.x, event.y)
        if not item and event.widget.__class__.__name__ == "Treeview":
            destroy_active_editor()


# Function for editing cell
def edit_cell(event):
    
    global active_editor, main_tree, tickers, comparison

    if main_tree.identify_region(event.x, event.y) == "cell":

        item_id = main_tree.focus()

        if not item_id:  # No item focused
            return

        column_id = main_tree.identify_column(event.x)
        col_index = main_tree.identify_row(event.y)

        # Retrieve the current value of the cell
        current_value = main_tree.item(col_index, 'values')[int(column_id[1:]) - 1]

        # Determine if this column should use a dropdown or text editor
        heading = main_tree.heading(column_id, 'text')

        if heading == "Comparison":
            options = [""] + comparison

        elif "Coin" in heading:
            options = [""] + tickers
        else:
            options = False

        if options:
            entry = create_dropdown_editor(item_id, column_id, current_value, options)
        else:
            entry = create_entry_editor(item_id, column_id, current_value)

        active_editor = [entry]
    

# Function for creating select dropdown
def create_dropdown_editor(item_id, column_id, current_value, options):
    global main_tree, root

    # Get the bounding box of the cell
    bbox = main_tree.bbox(item_id, column_id)
    if not bbox:
        return

    if not options:
        return
    
    x, y, width, height = bbox

    # Define dropdown options
    selected_option = tk.StringVar(root)
    selected_option.set(current_value if current_value in options else options[0])

    # Create OptionMenu
    editor = ttk.OptionMenu(main_tree, selected_option, *options, command=lambda val: save_edit(item_id, column_id, val))
    editor.place(x=x, y=y, width=width, height=height)
    editor.focus_set()

    return editor


# Function for creating input text box
def create_entry_editor(item_id, column_id, current_value):
    global main_tree

    # Get the bounding box of the cell
    bbox = main_tree.bbox(item_id, column_id)
    if not bbox:
        return

    x, y, width, height = bbox

    # Create Entry widget
    editor = ttk.Entry(main_tree, width=width // 8)
    editor.insert(0, current_value)
    editor.place(x=x, y=y, width=width, height=height)
    editor.focus_set()
    editor.bind("<Return>", lambda e: save_edit(item_id, column_id, editor.get()))
    editor.bind("<FocusOut>", lambda e: save_edit(item_id, column_id, editor.get()))

    return editor


# Function for saving entries change
def save_edit(item_id, column_id, new_value):
    global main_tree

    # Update the Treeview cell with the new value
    main_tree.set(item_id, column_id, new_value)


# Function for saving all the rows data
def save_rows():
    global main_tree, worker_config

    destroy_active_editor()

    main_tree.focus_set()

    data = []
    for item_id in main_tree.get_children():
        values = main_tree.item(item_id, 'values')

        # Refusing to save empty row
        if all(not x for x in values):
            continue

        if not validate_email(values[0]):
            messagebox.showerror("Validation Result", "Saving Failed due to invalid email format.")
            return "Invalid Email"
        
        if not validate_ticker(values[1]):
            messagebox.showerror("Validation Result", "Saving Failed due to invalid source coin ticker.")
            return "Invalid source coin"
        
        if not validate_ticker(values[2]):
            messagebox.showerror("Validation Result", "Saving Failed due to invalid target coin ticker.")
            return "Invalid target coin"
        
        if not validate_decimal_string(values[3]):
            messagebox.showerror("Validation Result", "Saving Failed due to invalid source value.")
            return "Invalid source value"
        
        if not validate_absolute_float(float(values[3])):
            messagebox.showerror("Validation Result", "Saving Failed due to invalid source value.")
            return "Invalid source value"

        if not validate_decimal_string(values[4]):
            messagebox.showerror("Validation Result", "Saving Failed due to invalid target valuea.")
            return "Invalid target value"
        
        if not validate_absolute_float(float(values[4])):
            messagebox.showerror("Validation Result", "Saving Failed due to invalid target value.")
            return "Invalid target value"
        
        if not validate_comparison(values[5]):
            messagebox.showerror("Validation Result", "Saving Failed due to invalid comparison value.")
            return "Invalid comparison"
        
        if not validate_numerical_string(values[6]):
            messagebox.showerror("Validation Result", "Saving Failed due to invalid email sent count.")
            return "Invalid email sent count"
        
        if not validate_positive_integer(int(values[6])):
            print("values:", values[6])
            messagebox.showerror("Validation Result", "Saving Failed due to invalid email sent count.")
            return "Invalid email sent count"

        item_data = {
            'email': values[0], 
            'source_coin': values[1], 
            'target_coin': values[2], 
            'source_value': float(values[3]), 
            'target_value': float(values[4]), 
            'comparison': values[5], 
            'email_sent_count': int(values[6]) 
        }

        data.append(item_data)

    worker_config['jobs'] = data

    with open('config.json', 'w') as f:
        json.dump(worker_config, f, indent=4)
        messagebox.showinfo("", "Save Completed")


# Function for deleting row(s)
def delete_row():
    global main_tree

    destroy_active_editor()

    selected_items = main_tree.selection()
    if selected_items:
        for item in selected_items:
            main_tree.delete(item)


# Function for adding a single row
def add_row():
    global main_tree, columns

    destroy_active_editor()
    
    main_tree.focus_set()

    i = len(main_tree.get_children())

    main_tree.insert("", "end", values=['' for col in columns], tags=("oddrow" if i % 2 == 0 else "evenrow"))


# Function for calling action command
def action_command(command, showMessage=False):
    global ui_config

    if ui_config['actions'] and ui_config['actions']['enable'] and ui_config['actions'][command]:
        subprocess.call(ui_config['actions'][command], shell=True)
        if showMessage:
            messagebox.showinfo("", showMessage)


# Function for decorate or styling the treeview
def decorate_styling():
    style = ttk.Style()
    style.theme_use("clam")
    style.configure("Treeview", rowheight=40) 
    style.configure("Treeview.Heading", font=("Helvetica", 11, "bold"))


# Function for building the main_tree
def build_tree():
    global main_tree, root, columns

    columns = ('email', 'source_coin', 'target_coin', 'source_value', 'target_value', 'comparison', 'email_sent_count')

    main_tree = ttk.Treeview(root, columns=columns, show="headings")

    main_tree.tag_configure("oddrow", background="#FAFAFA")
    main_tree.tag_configure("evenrow", background="#F7F7F7")

    main_tree.pack(fill="both", expand=True, padx=5, pady=5)
    main_tree.bind("<Double-1>", lambda event: edit_cell(event))
    main_tree.bind("<<TreeviewSelect>>", lambda event: destroy_active_editor())


# Function for building rows
def build_rows(showMessage=False):
    global rows, main_tree, columns, worker_config

    with open('config.json', 'r') as f:
        worker_config = json.load(f)
        rows = worker_config['jobs']

    for row in main_tree.get_children():
        main_tree.delete(row)

    # Build Rows
    for i, item in enumerate(rows):
        main_tree.insert("", "end", values=[item[col] for col in columns], tags=("oddrow" if i % 2 == 0 else "evenrow"))

    if showMessage:
        messagebox.showinfo("", showMessage)


# Function for building headings
def build_headings():
    global main_tree

    main_tree.heading("email", text="Email")
    main_tree.heading("source_coin", text="Source Coin")
    main_tree.heading("target_coin", text="Target Coin")
    main_tree.heading("source_value", text="Source Value")
    main_tree.heading("target_value", text="Target Value")
    main_tree.heading("comparison", text="Comparison")
    main_tree.heading("email_sent_count", text="Email Sent Count")


# function for building Buttons
def build_buttons():
    global root

    button_frame = tk.Frame(root, borderwidth=2)
    button_frame.pack()

    add_button = tk.Button(button_frame, text="Add", command=lambda: add_row())
    add_button.pack(side=tk.LEFT, padx=5, pady=5)

    delete_button = tk.Button(button_frame, text="Delete", command=lambda: delete_row())
    delete_button.pack(side=tk.LEFT, padx=5, pady=5)

    save_button = tk.Button(button_frame, text="Save", command=lambda: save_rows())
    save_button.pack(side=tk.LEFT, padx=5, pady=5)

    reload_button = tk.Button(button_frame, text="Reload", command=lambda: build_rows("Data Reloaded"))
    reload_button.pack(side=tk.LEFT, padx=5, pady=5)

    if ui_config['actions'] and ui_config['actions']['enable'] and ui_config['actions']['push']:
        push_button = tk.Button(button_frame, text="Upload to server", command=lambda: action_command('push', "Upload to server completed"))
        push_button.pack(side=tk.LEFT, padx=5, pady=5)

    if ui_config['actions'] and ui_config['actions']['enable'] and ui_config['actions']['pull']:
        pull_button = tk.Button(button_frame, text="Retrieve from server", command=lambda: action_command('pull', "Retrieving data completed, You may wish to reload the data"))
        pull_button.pack(side=tk.LEFT, padx=5, pady=5)


# Load the config
def load_config():
    global ui_config

    with open('configui.json', 'r') as f:
        ui_config = json.load(f)


# Function for validating email
def validate_email(email):
    # A basic regex pattern for email validation
    email_pattern = r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
    if re.match(email_pattern, email):
        return True
    return False


# Validation function for ticker value
def validate_ticker(ticker):
    global tickers
    return ticker in tickers


# Validation function for absolute float number, no 0 allowed
def validate_absolute_float(value):
    return isinstance(value, float) and value > 0


# Validation function for operator defined in comparison
def validate_comparison(operator):
    global comparison
    return operator in comparison


# Validation function for positive integer with 0 allowed
def validate_positive_integer(value):
    return isinstance(value, int) and value >= 0


# Validation to check if is numerical string
def validate_numerical_string(value):
    return value.isnumeric()


# Validation to check if is decimal string
def validate_decimal_string(value):
    try:
        float(value)
        return True
    except ValueError:
        return False
    


# Main function
def main():

    global root

    root = tk.Tk()
    root.title("Manage crypto checker jobs")
    
    # Loading the config
    load_config()

    # Style the table
    decorate_styling()

    # Build the main treeview
    build_tree()

    # Build headings
    build_headings()

    # Build Rows
    build_rows()

    # Build Buttons
    build_buttons()

    # Global event for closing active editor when user clicked outside the element
    root.bind("<Button-1>", lambda event: click_outside_item_callback(event))

    root.mainloop()

# Boot the UI
if __name__ == '__main__':
    main()
