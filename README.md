# Mailflow
#### 🚀 **Customer Support Email Automation with AI Agents and RAG (in Go lang)**


-----
Note: Last stable code version is available at branch 0.9: [Click Here](https://github.com/zaynkorai/mailflow/tree/0.9)

## **Introduction**

This AI-powered solution, built with a Go-driven workflow, uses multiple AI agents to efficiently manage, categorize, and respond to customer emails. It also leverages Retrieval-Augmented Generation (RAG) technology to provide accurate answers to any business or product questions, ensuring your customers always get the precise information they need.

-----
<img width="1138" alt="working output" src="https://github.com/user-attachments/assets/9ba471f2-0c63-4adf-9643-2950c303d435" />


## Features

### Email Inbox Management with AI Agents

  * **Continuously monitors** the agency's Gmail inbox
  * **Categorizes emails** into '**customer complaint**,' '**product inquiry**,' '**customer feedback**,' or '**unrelated**'
  * **Automatically handles irrelevant emails** to maintain efficiency

### AI Response Generation

  * **Quickly drafts emails** for customer complaints and feedback using a customizable workflow
  * **Utilizes RAG techniques** to answer product/service-related questions accurately
  * **Creates personalized email content** tailored to each customer's needs

### **Quality Assurance with AI**

  * **Automatically checks** email quality, formatting, and relevance
  * **Ensures every response** meets high standards before reaching the client

-----

## **How It Works**

1.  **Email Monitoring**: The system constantly checks for new emails in the agency's Gmail inbox using the Gmail API.
2.  **Email Categorization**: AI agents sort each email into predefined categories.
3.  **Response Generation**:
      * **For complaints or feedback**: The system quickly drafts a tailored email response.
      * **For service/product questions**: The system uses RAG to retrieve accurate information from agency documents and generates a response.
4.  **Quality Assurance**: Each draft email undergoes AI quality and formatting checks.
5.  **Sending**: Approved emails are sent to the client promptly, ensuring timely communication.

-----

## System Flowchart

This is the detailed flow of the system:


-----

## Tech Stack

  * **Custom Graph Implementation (Go)**: For developing the AI agents workflow, replacing Langchain & Langgraph.
  * **Google Gemini API**: For large language model (LLM) access and embeddings.
  * **Google Gmail API**: For email inbox management.

-----

## How to Run

### Prerequisites

  * **Go** (version 1.22+)
  * **Google Gemini API Key**
  * **Gmail API credentials**

### Setup

1.  **Clone the repository (or set up your Go module):**

    ```sh
    git clone https://github.com/zaynkorai/mailflow.git 
    cd mailflow
    ```

3.  **Set up environment variables:**

    Create a `.env` file in the root directory of your project and add your Gmail address and Google Gemini API key:

    ```env
    PORT=8080
    MY_EMAIL=your_email@gmail.com
    GOOGLE_API_KEY=your_gemini_api_key
    ```

4.  **Ensure Gmail API is enabled:**

    Follow [this guide](https://developers.google.com/gmail/api/quickstart/python) to enable the Gmail API for your Google Cloud project and obtain your `credentials.json` file. Place `credentials.json` in your project's root directory.

### Running the Application

1.  **Indexing RAG (console application):**

    Fill out agency/company data in `data/agency.txt` and run given command

    ```sh
    go run data/indexing.go
    ```


2.  **Start the workflow (console application):**

    ```sh
    go run main.go
    ```

    The application will start checking for new emails, categorizing them, synthesizing queries, drafting responses, and verifying email quality, logging progress to your console.


-----

### Contributing

Contributions are welcome\! Please open an issue or submit a pull request for any changes.


### Contact

If you have any questions or suggestions, feel free to contact me at `zaynulabdin313@gmail.com`.


-----
